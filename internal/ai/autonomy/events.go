package autonomy

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"cyberos/internal/contracts"
)

// AutonomyEventContractV1 is the schema version for autonomy domain events (Phase 15A).
const AutonomyEventContractV1 contracts.SchemaVersion = "1"

// Payload size bounds (fail closed on oversized untrusted data).
const (
	MaxAutonomyEventPayloadBytes = 64 * 1024
	MaxAutonomyEventStringLen    = 512
	MaxIdempotencyKeyLen         = 256
)

// AutonomyEventType is the taxonomy of autonomy-related domain events.
// Events are informational only — never authorization or execution.
type AutonomyEventType string

const (
	EventExecutionCompleted    AutonomyEventType = "execution.completed"
	EventExecutionFailed       AutonomyEventType = "execution.failed"
	EventFindingCreated        AutonomyEventType = "finding.created"
	EventWorkflowCompleted     AutonomyEventType = "workflow.completed"
	EventWorkflowFailed        AutonomyEventType = "workflow.failed"
	EventAutonomyPolicyBlocked AutonomyEventType = "autonomy.policy_blocked"
)

// Valid reports whether t is a known autonomy event type.
func (t AutonomyEventType) Valid() bool {
	switch t {
	case EventExecutionCompleted, EventExecutionFailed, EventFindingCreated,
		EventWorkflowCompleted, EventWorkflowFailed, EventAutonomyPolicyBlocked:
		return true
	default:
		return false
	}
}

// AutonomyEvent is a versioned, project-scoped domain event for future observe loops.
// It does not authorize, issue grants, or invoke Orchestrator/Broker/Executor.
type AutonomyEvent struct {
	EventID        string                  `json:"event_id"`
	SchemaVersion  contracts.SchemaVersion `json:"schema_version"`
	Type           AutonomyEventType       `json:"type"`
	ProjectID      contracts.ProjectID     `json:"project_id"`
	TenantID       contracts.TenantID      `json:"tenant_id,omitempty"`
	ScopeID        contracts.ScopeID       `json:"scope_id,omitempty"`
	AutonomyRunID  string                  `json:"autonomy_run_id,omitempty"`
	WorkflowRunID  string                  `json:"workflow_run_id,omitempty"`
	ExecutionID    string                  `json:"execution_id,omitempty"`
	CorrelationID  contracts.CorrelationID `json:"correlation_id,omitempty"`
	IdempotencyKey string                  `json:"idempotency_key"`
	Timestamp      time.Time               `json:"timestamp"`
	// Payload is untrusted bounded metadata (never secrets, argv, or grants).
	Payload map[string]any `json:"payload,omitempty"`
}

// IdempotencyIdentity is the logical dedupe key scope (project + key).
// Does not authorize anything.
func (e AutonomyEvent) IdempotencyIdentity() string {
	return string(e.ProjectID) + "|" + e.IdempotencyKey
}

// ToEnvelope maps to the existing EventEnvelope for Hub/SSE compatibility.
func (e AutonomyEvent) ToEnvelope() contracts.EventEnvelope {
	payload := map[string]any{}
	if e.Payload != nil {
		for k, v := range e.Payload {
			payload[k] = v
		}
	}
	payload["autonomy_run_id"] = e.AutonomyRunID
	payload["workflow_run_id"] = e.WorkflowRunID
	payload["execution_id"] = e.ExecutionID
	payload["idempotency_key"] = e.IdempotencyKey
	if e.TenantID != "" {
		payload["tenant_id"] = string(e.TenantID)
	}
	return contracts.EventEnvelope{
		ID:            contracts.EventID(e.EventID),
		Type:          contracts.EventType(e.Type),
		SchemaVersion: e.SchemaVersion,
		Timestamp:     e.Timestamp,
		ProjectID:     e.ProjectID,
		ScopeID:       e.ScopeID,
		CorrelationID: e.CorrelationID,
		Payload:       payload,
	}
}

// ValidateAutonomyEvent fail-closed validation for Phase 15A contracts.
func ValidateAutonomyEvent(e AutonomyEvent) error {
	if e.SchemaVersion == "" {
		return fmt.Errorf("%w: missing schema_version", contracts.ErrInvalidArgument)
	}
	if e.SchemaVersion != AutonomyEventContractV1 {
		return fmt.Errorf("%w: unsupported autonomy event schema %q", contracts.ErrInvalidArgument, e.SchemaVersion)
	}
	if err := contracts.ValidateNonEmpty("event_id", e.EventID); err != nil {
		return err
	}
	if utf8.RuneCountInString(e.EventID) > MaxAutonomyEventStringLen {
		return fmt.Errorf("%w: event_id too long", contracts.ErrInvalidArgument)
	}
	if !e.Type.Valid() {
		return fmt.Errorf("%w: invalid autonomy event type %q", contracts.ErrInvalidArgument, e.Type)
	}
	if err := contracts.ValidateNonEmpty("project_id", string(e.ProjectID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("idempotency_key", e.IdempotencyKey); err != nil {
		return err
	}
	if utf8.RuneCountInString(e.IdempotencyKey) > MaxIdempotencyKeyLen {
		return fmt.Errorf("%w: idempotency_key too long", contracts.ErrInvalidArgument)
	}
	if strings.TrimSpace(e.IdempotencyKey) != e.IdempotencyKey {
		return fmt.Errorf("%w: idempotency_key must not have leading/trailing space", contracts.ErrInvalidArgument)
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("%w: timestamp required", contracts.ErrInvalidArgument)
	}
	if err := validatePayloadBounds(e.Payload); err != nil {
		return err
	}
	return nil
}

func validatePayloadBounds(p map[string]any) error {
	if p == nil {
		return nil
	}
	// Rough size: keys + string values; reject if too many entries or long strings.
	if len(p) > 64 {
		return fmt.Errorf("%w: payload too many keys", contracts.ErrInvalidArgument)
	}
	approx := 0
	for k, v := range p {
		if utf8.RuneCountInString(k) > MaxAutonomyEventStringLen {
			return fmt.Errorf("%w: payload key too long", contracts.ErrInvalidArgument)
		}
		approx += len(k)
		switch t := v.(type) {
		case string:
			if utf8.RuneCountInString(t) > MaxAutonomyEventStringLen {
				return fmt.Errorf("%w: payload value too long", contracts.ErrInvalidArgument)
			}
			approx += len(t)
			if looksLikeSecretKey(k) {
				return fmt.Errorf("%w: payload must not contain secret field %q", contracts.ErrInvalidArgument, k)
			}
		case nil, bool, float64, int, int64:
			approx += 16
		default:
			// nested structures: reject to keep payload simple/bounded
			return fmt.Errorf("%w: payload values must be scalar", contracts.ErrInvalidArgument)
		}
	}
	if approx > MaxAutonomyEventPayloadBytes {
		return fmt.Errorf("%w: payload too large", contracts.ErrInvalidArgument)
	}
	return nil
}

func looksLikeSecretKey(k string) bool {
	lk := strings.ToLower(k)
	for _, s := range []string{"api_key", "apikey", "password", "secret", "token", "private_key", "bearer", "credential"} {
		if strings.Contains(lk, s) {
			return true
		}
	}
	return false
}

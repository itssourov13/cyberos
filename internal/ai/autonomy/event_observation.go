package autonomy

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// ObservationContractV1 is the schema version for event-derived observations (Phase 15A).
// Distinct from Phase 13 ObservationRequest (Analyze input).
const ObservationContractV1 contracts.SchemaVersion = "1"

// ObservationSource identifies the origin of an event-derived observation.
type ObservationSource string

const (
	ObservationSourceExecution ObservationSource = "execution"
	ObservationSourceFinding   ObservationSource = "finding"
	ObservationSourceWorkflow  ObservationSource = "workflow"
	ObservationSourcePolicy    ObservationSource = "policy"
)

// EventObservationRequest is the pure contract produced from a validated AutonomyEvent.
// Phase 15A STOP boundary: no Brain, Policy, Handoff, or execution.
type EventObservationRequest struct {
	ContractVersion  contracts.SchemaVersion `json:"contract_version"`
	EventID          string                  `json:"event_id"`
	IdempotencyKey   string                  `json:"idempotency_key"`
	ProjectID        contracts.ProjectID     `json:"project_id"`
	TenantID         contracts.TenantID      `json:"tenant_id,omitempty"`
	ScopeID          contracts.ScopeID       `json:"scope_id,omitempty"`
	AutonomyRunID    string                  `json:"autonomy_run_id,omitempty"`
	WorkflowRunID    string                  `json:"workflow_run_id,omitempty"`
	ExecutionID      string                  `json:"execution_id,omitempty"`
	CorrelationID    contracts.CorrelationID `json:"correlation_id,omitempty"`
	TriggerType      AutonomyEventType       `json:"trigger_type"`
	TriggerTimestamp time.Time               `json:"trigger_timestamp"`
	Source           ObservationSource       `json:"source"`
}

// SourceFromEventType maps event type to observation source.
func SourceFromEventType(t AutonomyEventType) ObservationSource {
	switch t {
	case EventExecutionCompleted, EventExecutionFailed:
		return ObservationSourceExecution
	case EventFindingCreated:
		return ObservationSourceFinding
	case EventWorkflowCompleted, EventWorkflowFailed:
		return ObservationSourceWorkflow
	case EventAutonomyPolicyBlocked:
		return ObservationSourcePolicy
	default:
		return ""
	}
}

// ToEventObservation converts a validated AutonomyEvent into an EventObservationRequest.
// Side-effect free; does not invoke Brain, Policy, Handoff, Orchestrator, Grant, or Broker.
func ToEventObservation(e AutonomyEvent) (EventObservationRequest, error) {
	if err := ValidateAutonomyEvent(e); err != nil {
		return EventObservationRequest{}, err
	}
	src := SourceFromEventType(e.Type)
	if src == "" {
		return EventObservationRequest{}, fmt.Errorf("%w: cannot map event type to observation source", contracts.ErrInvalidArgument)
	}
	return EventObservationRequest{
		ContractVersion:  ObservationContractV1,
		EventID:          e.EventID,
		IdempotencyKey:   e.IdempotencyKey,
		ProjectID:        e.ProjectID,
		TenantID:         e.TenantID,
		ScopeID:          e.ScopeID,
		AutonomyRunID:    e.AutonomyRunID,
		WorkflowRunID:    e.WorkflowRunID,
		ExecutionID:      e.ExecutionID,
		CorrelationID:    e.CorrelationID,
		TriggerType:      e.Type,
		TriggerTimestamp: e.Timestamp,
		Source:           src,
	}, nil
}

// ValidateEventObservation fail-closed check for observation contract.
func ValidateEventObservation(o EventObservationRequest) error {
	if o.ContractVersion == "" {
		return fmt.Errorf("%w: missing observation contract_version", contracts.ErrInvalidArgument)
	}
	if o.ContractVersion != ObservationContractV1 {
		return fmt.Errorf("%w: unsupported observation contract %q", contracts.ErrInvalidArgument, o.ContractVersion)
	}
	if err := contracts.ValidateNonEmpty("event_id", o.EventID); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("idempotency_key", o.IdempotencyKey); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("project_id", string(o.ProjectID)); err != nil {
		return err
	}
	if !o.TriggerType.Valid() {
		return fmt.Errorf("%w: invalid trigger_type", contracts.ErrInvalidArgument)
	}
	if o.Source == "" {
		return fmt.Errorf("%w: source required", contracts.ErrInvalidArgument)
	}
	if o.TriggerTimestamp.IsZero() {
		return fmt.Errorf("%w: trigger_timestamp required", contracts.ErrInvalidArgument)
	}
	return nil
}

// SameLogicalEvent reports whether two events share project-scoped idempotency identity.
func SameLogicalEvent(a, b AutonomyEvent) bool {
	if a.ProjectID == "" || a.IdempotencyKey == "" {
		return false
	}
	return a.IdempotencyIdentity() == b.IdempotencyIdentity()
}

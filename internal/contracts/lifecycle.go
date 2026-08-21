package contracts

import "time"

// ExecutionLifecycleState is the tracked state of a single execution attempt.
type ExecutionLifecycleState string

const (
	ExecutionLifecycleQueued    ExecutionLifecycleState = "QUEUED"
	ExecutionLifecycleStarted   ExecutionLifecycleState = "STARTED"
	ExecutionLifecycleSucceeded ExecutionLifecycleState = "SUCCEEDED"
	ExecutionLifecycleFailed    ExecutionLifecycleState = "FAILED"
	ExecutionLifecycleTimeout   ExecutionLifecycleState = "TIMEOUT"
	ExecutionLifecycleRejected  ExecutionLifecycleState = "REJECTED"
	ExecutionLifecycleCancelled ExecutionLifecycleState = "CANCELLED"
)

// Valid reports whether the lifecycle state is known.
func (s ExecutionLifecycleState) Valid() bool {
	switch s {
	case ExecutionLifecycleQueued, ExecutionLifecycleStarted, ExecutionLifecycleSucceeded, ExecutionLifecycleFailed,
		ExecutionLifecycleTimeout, ExecutionLifecycleRejected, ExecutionLifecycleCancelled:
		return true
	default:
		return false
	}
}

// ExecutionRecord is an attributable record of one execution attempt.
// It correlates Task → Grant → Execution and is suitable for audit/evidence.
type ExecutionRecord struct {
	ID            ExecutionID             `json:"id"`
	State         ExecutionLifecycleState `json:"state"`
	TaskID        TaskID                  `json:"task_id"`
	GrantID       ExecutionGrantID        `json:"grant_id"`
	Principal     PrincipalRef            `json:"principal"`
	ProjectID     ProjectID               `json:"project_id"`
	ScopeID       ScopeID                 `json:"scope_id"`
	CapabilityID  CapabilityID            `json:"capability_id"`
	Attempt       int                     `json:"attempt"`
	StartedAt     Timestamp               `json:"started_at"`
	FinishedAt    Timestamp               `json:"finished_at,omitempty"`
	Duration      time.Duration           `json:"duration,omitempty"`
	Outcome       ExecutionStatus         `json:"outcome,omitempty"`
	FailureReason string                  `json:"failure_reason,omitempty"`
	ExecutorID    string                  `json:"executor_id,omitempty"`
	CorrelationID  CorrelationID           `json:"correlation_id,omitempty"`
	CausationID    string                  `json:"causation_id,omitempty"`
	IdempotencyKey string                  `json:"idempotency_key,omitempty"`
	ToolID         string                  `json:"tool_id,omitempty"`
	PackVersion    string                  `json:"pack_version,omitempty"`
	TenantID       TenantID                `json:"tenant_id,omitempty"`
	// ResourcePolicySnapshot is a copy of limits applied for this attempt.
	ResourcePolicySnapshot *ResourcePolicy `json:"resource_policy,omitempty"`
	// Metrics are safe, non-host-identifying execution metrics.
	Metrics map[string]any `json:"metrics,omitempty"`
}

// Validate checks required correlation fields.
func (r ExecutionRecord) Validate() error {
	if err := ValidateNonEmpty("execution.id", string(r.ID)); err != nil {
		return err
	}
	if !r.State.Valid() {
		return ErrInvalidArgument
	}
	if err := ValidateNonEmpty("execution.task_id", string(r.TaskID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("execution.grant_id", string(r.GrantID)); err != nil {
		return err
	}
	if r.Principal.ID == "" {
		return ErrInvalidArgument
	}
	return nil
}

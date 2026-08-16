package contracts

// ExecutionRequest is the input presented to a Tool Broker / Executor.
// It must always carry a previously issued ExecutionGrant.
type ExecutionRequest struct {
	Grant ExecutionGrant `json:"grant"`
	// Parameters are capability-specific and opaque to the core contracts.
	Parameters map[string]any `json:"parameters,omitempty"`
	// CorrelationID should match or continue the grant's correlation chain.
	CorrelationID CorrelationID `json:"correlation_id,omitempty"`
}

// Validate checks that a grant is present and structurally valid.
// Full authorization and grant matching are the responsibility of the
// Tool Broker implementation (later phase).
func (r ExecutionRequest) Validate() error {
	return r.Grant.Validate()
}

// ExecutionStatus represents the outcome of an execution attempt.
type ExecutionStatus string

const (
	ExecutionStatusSucceeded ExecutionStatus = "SUCCEEDED"
	ExecutionStatusFailed    ExecutionStatus = "FAILED"
	ExecutionStatusTimeout   ExecutionStatus = "TIMEOUT"
	ExecutionStatusCancelled ExecutionStatus = "CANCELLED"
	ExecutionStatusRejected  ExecutionStatus = "REJECTED" // e.g. grant invalid
)

// Valid reports whether the status is known.
func (s ExecutionStatus) Valid() bool {
	switch s {
	case ExecutionStatusSucceeded, ExecutionStatusFailed,
		ExecutionStatusTimeout, ExecutionStatusCancelled, ExecutionStatusRejected:
		return true
	default:
		return false
	}
}

// ExecutionResult is the structured output of an executor.
// Evidence and findings are produced from this result in later phases.
type ExecutionResult struct {
	Status     ExecutionStatus `json:"status"`
	StartedAt  Timestamp       `json:"started_at"`
	FinishedAt Timestamp       `json:"finished_at"`
	// Output is opaque, capability-specific data. Prefer structured maps.
	Output        map[string]any `json:"output,omitempty"`
	ErrorMessage  string         `json:"error_message,omitempty"`
	CorrelationID CorrelationID  `json:"correlation_id,omitempty"`
	// ExecutorID identifies which executor implementation produced the result
	// (e.g. "mock", "sandbox-go", future plugin names).
	ExecutorID string `json:"executor_id,omitempty"`
}

// Validate performs basic structural checks.
func (r ExecutionResult) Validate() error {
	if !r.Status.Valid() {
		return ErrInvalidArgument
	}
	if r.StartedAt.IsZero() || r.FinishedAt.IsZero() {
		return ErrInvalidArgument
	}
	return nil
}

// Executor is the contract boundary for any component that can perform work
// under a valid ExecutionGrant.
//
// Future implementations may include:
//   - safe mock executor (Phase 1)
//   - internal Go tools
//   - isolated language plugins (Python, Rust, …)
//   - remote / sandboxed workers
//
// The core never calls os/exec, shell, or real offensive tools directly.
// All execution paths must go through a grant-validated Executor.
type Executor interface {
	// Execute runs the requested work under the supplied grant.
	// Implementations MUST reject requests with invalid, expired, or
	// mismatched grants.
	Execute(req ExecutionRequest) (ExecutionResult, error)
}

// ToolBroker is the sole execution gateway of CYBEROS.
// It validates the grant and routes to an appropriate Executor.
// Phase 1A only defines the interface; the implementation arrives later.
type ToolBroker interface {
	// Execute validates the grant and dispatches to an executor.
	Execute(req ExecutionRequest) (ExecutionResult, error)
}

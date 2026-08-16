package contracts

// TaskState represents the explicit lifecycle of a task.
// Transitions must be validated; arbitrary mutation is forbidden.
type TaskState string

const (
	TaskStateCreated         TaskState = "CREATED"
	TaskStatePending         TaskState = "PENDING"
	TaskStateAuthorized      TaskState = "AUTHORIZED"
	TaskStateWaitingApproval TaskState = "WAITING_APPROVAL"
	TaskStateReady           TaskState = "READY"
	TaskStateRunning         TaskState = "RUNNING"
	TaskStateSucceeded       TaskState = "SUCCEEDED"
	TaskStateFailed          TaskState = "FAILED"
	TaskStateCancelled       TaskState = "CANCELLED"
	TaskStateExpired         TaskState = "EXPIRED"
)

// Valid reports whether the state is known.
func (s TaskState) Valid() bool {
	switch s {
	case TaskStateCreated, TaskStatePending, TaskStateAuthorized,
		TaskStateWaitingApproval, TaskStateReady, TaskStateRunning,
		TaskStateSucceeded, TaskStateFailed, TaskStateCancelled, TaskStateExpired:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether the state is a final state.
func (s TaskState) IsTerminal() bool {
	switch s {
	case TaskStateSucceeded, TaskStateFailed, TaskStateCancelled, TaskStateExpired:
		return true
	default:
		return false
	}
}

// TaskType is a coarse classification of the kind of work.
type TaskType string

const (
	TaskTypeRecon      TaskType = "RECON"
	TaskTypeScan       TaskType = "SCAN"
	TaskTypeAnalysis   TaskType = "ANALYSIS"
	TaskTypeValidation TaskType = "VALIDATION"
	TaskTypeReport     TaskType = "REPORT"
	TaskTypeCustom     TaskType = "CUSTOM"
)

// TaskRef is a lightweight reference to a task.
type TaskRef struct {
	ID        TaskID    `json:"id"`
	ProjectID ProjectID `json:"project_id"`
	ScopeID   ScopeID   `json:"scope_id"`
	State     TaskState `json:"state"`
	Type      TaskType  `json:"type,omitempty"`
}

// Validate checks required fields and state.
func (t TaskRef) Validate() error {
	if err := ValidateNonEmpty("task.id", string(t.ID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("task.project_id", string(t.ProjectID)); err != nil {
		return err
	}
	if err := ValidateNonEmpty("task.scope_id", string(t.ScopeID)); err != nil {
		return err
	}
	if !t.State.Valid() {
		return ErrInvalidArgument
	}
	return nil
}

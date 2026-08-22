package workflow

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// RunState is the lifecycle of a workflow run (orchestration only).
type RunState string

const (
	RunStatePending   RunState = "PENDING"
	RunStateRunning   RunState = "RUNNING"
	RunStateSucceeded RunState = "SUCCEEDED"
	RunStateFailed    RunState = "FAILED"
	RunStateCancelled RunState = "CANCELLED"
)

// Valid reports whether the run state is known.
func (s RunState) Valid() bool {
	switch s {
	case RunStatePending, RunStateRunning, RunStateSucceeded, RunStateFailed, RunStateCancelled:
		return true
	default:
		return false
	}
}

// StepRunState tracks a single step within a run.
type StepRunState string

const (
	StepPending   StepRunState = "PENDING"
	StepReady     StepRunState = "READY"
	StepRunning   StepRunState = "RUNNING"
	StepSucceeded StepRunState = "SUCCEEDED"
	StepFailed    StepRunState = "FAILED"
	StepSkipped   StepRunState = "SKIPPED"
	StepCancelled StepRunState = "CANCELLED"
)

// StepStatus is the attribution for one workflow step.
type StepStatus struct {
	StepID       string
	TaskID       contracts.TaskID
	CapabilityID contracts.CapabilityID
	State        StepRunState
	ToolID       string
	ExecutionID  contracts.ExecutionID
	EvidenceID   contracts.EvidenceID
	Error        string
}

// Run is an instance of a workflow definition being orchestrated.
// It does not hold grants or authorization privileges.
type Run struct {
	ID            string
	WorkflowID    string
	Version       contracts.SchemaVersion
	ProjectID     contracts.ProjectID
	ScopeID       contracts.ScopeID
	PrincipalID   contracts.PrincipalID
	State         RunState
	StepToTask    map[string]contracts.TaskID
	Steps         map[string]*StepStatus
	CorrelationID contracts.CorrelationID
	StartedAt     time.Time
	FinishedAt    time.Time
	Error         string
}

// Validate checks required run fields.
func (r *Run) Validate() error {
	if r == nil {
		return fmt.Errorf("%w: run nil", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("run.id", r.ID); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("run.workflow_id", r.WorkflowID); err != nil {
		return err
	}
	if !r.State.Valid() {
		return fmt.Errorf("%w: invalid run state", contracts.ErrInvalidArgument)
	}
	return nil
}

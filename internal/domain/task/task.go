// Package task implements the Task domain entity and its explicit state machine.
package task

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Task is a unit of work bound to a project, scope, principal, and capability.
type Task struct {
	ID           contracts.TaskID
	ProjectID    contracts.ProjectID
	ScopeID      contracts.ScopeID
	PrincipalID  contracts.PrincipalID
	CapabilityID contracts.CapabilityID
	Type         contracts.TaskType
	State        contracts.TaskState
	Title        string
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Metadata     map[string]string
}

// New creates a Task in CREATED state or returns an error.
func New(
	id contracts.TaskID,
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
	principalID contracts.PrincipalID,
	capabilityID contracts.CapabilityID,
	typ contracts.TaskType,
	title string,
) (*Task, error) {
	if err := contracts.ValidateNonEmpty("task.id", string(id)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("task.project_id", string(projectID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("task.scope_id", string(scopeID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("task.principal_id", string(principalID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("task.capability_id", string(capabilityID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("task.title", title); err != nil {
		return nil, err
	}
	now := contracts.NowUTC()
	return &Task{
		ID:           id,
		ProjectID:    projectID,
		ScopeID:      scopeID,
		PrincipalID:  principalID,
		CapabilityID: capabilityID,
		Type:         typ,
		State:        contracts.TaskStateCreated,
		Title:        title,
		CreatedAt:    now,
		UpdatedAt:    now,
		Metadata:     make(map[string]string),
	}, nil
}

// Validate checks structural invariants.
func (t *Task) Validate() error {
	if t == nil {
		return fmt.Errorf("%w: task is nil", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("task.id", string(t.ID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("task.project_id", string(t.ProjectID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("task.scope_id", string(t.ScopeID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("task.principal_id", string(t.PrincipalID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("task.capability_id", string(t.CapabilityID)); err != nil {
		return err
	}
	if !t.State.Valid() {
		return fmt.Errorf("%w: invalid task state %q", contracts.ErrInvalidArgument, t.State)
	}
	if err := contracts.ValidateNonEmpty("task.title", t.Title); err != nil {
		return err
	}
	return nil
}

// allowedTransitions defines the explicit state machine.
// Future phases may extend this map; illegal transitions must always fail.
var allowedTransitions = map[contracts.TaskState][]contracts.TaskState{
	contracts.TaskStateCreated: {
		contracts.TaskStatePending,
		contracts.TaskStateCancelled,
		contracts.TaskStateExpired,
	},
	contracts.TaskStatePending: {
		contracts.TaskStateAuthorized,
		contracts.TaskStateWaitingApproval,
		contracts.TaskStateCancelled,
		contracts.TaskStateExpired,
	},
	contracts.TaskStateAuthorized: {
		contracts.TaskStateReady,
		contracts.TaskStateWaitingApproval,
		contracts.TaskStateCancelled,
		contracts.TaskStateExpired,
	},
	contracts.TaskStateWaitingApproval: {
		contracts.TaskStateAuthorized,
		contracts.TaskStateCancelled,
		contracts.TaskStateExpired,
	},
	contracts.TaskStateReady: {
		contracts.TaskStateRunning,
		contracts.TaskStateCancelled,
		contracts.TaskStateExpired,
	},
	contracts.TaskStateRunning: {
		contracts.TaskStateSucceeded,
		contracts.TaskStateFailed,
		contracts.TaskStateCancelled,
	},
	// Terminal states have no outbound transitions.
	contracts.TaskStateSucceeded: {},
	contracts.TaskStateFailed:    {},
	contracts.TaskStateCancelled: {},
	contracts.TaskStateExpired:   {},
}

// CanTransitionTo reports whether the transition is legal.
func (t *Task) CanTransitionTo(next contracts.TaskState) bool {
	if t == nil || !next.Valid() {
		return false
	}
	allowed, ok := allowedTransitions[t.State]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == next {
			return true
		}
	}
	return false
}

// TransitionTo moves the task to the next state if the transition is legal.
// Illegal transitions return ErrInvalidStateTransition.
func (t *Task) TransitionTo(next contracts.TaskState) error {
	if t == nil {
		return fmt.Errorf("%w: task is nil", contracts.ErrInvalidArgument)
	}
	if !t.CanTransitionTo(next) {
		return fmt.Errorf("%w: cannot transition from %s to %s",
			contracts.ErrInvalidStateTransition, t.State, next)
	}
	t.State = next
	t.UpdatedAt = contracts.NowUTC()
	return nil
}

// IsTerminal reports whether the task is in a final state.
func (t *Task) IsTerminal() bool {
	return t != nil && t.State.IsTerminal()
}

// Ref returns a lightweight contract reference.
func (t *Task) Ref() contracts.TaskRef {
	return contracts.TaskRef{
		ID:        t.ID,
		ProjectID: t.ProjectID,
		ScopeID:   t.ScopeID,
		State:     t.State,
		Type:      t.Type,
	}
}

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

	// Schedule holds optional scheduling metadata.
	// Presence of a schedule does NOT authorize the task.
	Schedule *ScheduleSpec

	// DependsOn lists task IDs that must reach SUCCEEDED before this task may run.
	// Dependencies are not authorization.
	DependsOn []contracts.TaskID

	// Retry holds optional retry policy and attempt tracking.
	Retry *RetryState

	// Timeout is the maximum execution duration. Zero means no limit at this layer.
	Timeout time.Duration

	// Deadline is an absolute execution deadline (optional).
	Deadline *time.Time
}

// RetryPolicy configures bounded retries. Zero MaxAttempts means no retries.
type RetryPolicy struct {
	MaxAttempts int           // total attempts allowed (>=1 means at least the first try)
	Backoff     time.Duration // fixed backoff between retries (Phase 2B simple)
}

// RetryState tracks attempts under a policy.
type RetryState struct {
	Policy    RetryPolicy
	Attempts  int
	LastError string
	NextRetry *time.Time
}

// ScheduleSpec describes when a task should become eligible for the pending queue.
// Scheduling is never authorization.
type ScheduleSpec struct {
	// NotBefore is the earliest time the task may leave SCHEDULED.
	NotBefore time.Time
	// Priority is a simple ordering hint (higher runs first among due tasks). Zero is fine.
	Priority int
	// QueuedAt is set when the scheduler promotes the task to PENDING.
	QueuedAt *time.Time
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
		contracts.TaskStateScheduled,
		contracts.TaskStatePending,
		contracts.TaskStateCancelled,
		contracts.TaskStateExpired,
	},
	contracts.TaskStateScheduled: {
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
	// FAILED may return to PENDING when retries remain (enforced by engine/policy).
	contracts.TaskStateFailed: {
		contracts.TaskStatePending,
	},
	// Terminal states (except FAILED, which is conditionally retriable).
	contracts.TaskStateSucceeded: {},
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

// AllowedNextStates returns legal next states from the current state.
func (t *Task) AllowedNextStates() []contracts.TaskState {
	if t == nil {
		return nil
	}
	out := allowedTransitions[t.State]
	cp := make([]contracts.TaskState, len(out))
	copy(cp, out)
	return cp
}

// EnsureBindings reports an error if project/scope/principal/capability bindings are incomplete.
func (t *Task) EnsureBindings() error {
	if t == nil {
		return fmt.Errorf("%w: task is nil", contracts.ErrInvalidArgument)
	}
	return t.Validate()
}

// ApplySchedule attaches schedule metadata. Does not change state by itself.
func (t *Task) ApplySchedule(spec ScheduleSpec) error {
	if t == nil {
		return fmt.Errorf("%w: task is nil", contracts.ErrInvalidArgument)
	}
	if spec.NotBefore.IsZero() {
		return fmt.Errorf("%w: schedule.not_before is required", contracts.ErrInvalidArgument)
	}
	cp := spec
	t.Schedule = &cp
	t.UpdatedAt = contracts.NowUTC()
	return nil
}

// IsDue reports whether a scheduled task is ready to be promoted at now.
func (t *Task) IsDue(now time.Time) bool {
	if t == nil || t.Schedule == nil {
		return false
	}
	if t.State != contracts.TaskStateScheduled {
		return false
	}
	return !now.Before(t.Schedule.NotBefore)
}

// SetDependencies replaces the dependency list (no cycle check here — use dependency graph).
func (t *Task) SetDependencies(ids []contracts.TaskID) error {
	if t == nil {
		return fmt.Errorf("%w: task is nil", contracts.ErrInvalidArgument)
	}
	for _, id := range ids {
		if id == "" {
			return fmt.Errorf("%w: empty dependency id", contracts.ErrInvalidArgument)
		}
		if id == t.ID {
			return fmt.Errorf("%w: task cannot depend on itself", contracts.ErrInvalidArgument)
		}
	}
	cp := make([]contracts.TaskID, len(ids))
	copy(cp, ids)
	t.DependsOn = cp
	t.UpdatedAt = contracts.NowUTC()
	return nil
}

// ApplyRetryPolicy attaches a bounded retry policy.
func (t *Task) ApplyRetryPolicy(p RetryPolicy) error {
	if t == nil {
		return fmt.Errorf("%w: task is nil", contracts.ErrInvalidArgument)
	}
	if p.MaxAttempts < 1 {
		return fmt.Errorf("%w: retry.max_attempts must be >= 1", contracts.ErrInvalidArgument)
	}
	if p.MaxAttempts > 100 {
		return fmt.Errorf("%w: retry.max_attempts exceeds hard limit (100)", contracts.ErrInvalidArgument)
	}
	t.Retry = &RetryState{Policy: p}
	t.UpdatedAt = contracts.NowUTC()
	return nil
}

// SetTimeout sets a per-execution timeout. Negative values are rejected.
func (t *Task) SetTimeout(d time.Duration) error {
	if t == nil {
		return fmt.Errorf("%w: task is nil", contracts.ErrInvalidArgument)
	}
	if d < 0 {
		return fmt.Errorf("%w: timeout must be >= 0", contracts.ErrInvalidArgument)
	}
	t.Timeout = d
	t.UpdatedAt = contracts.NowUTC()
	return nil
}

// SetDeadline sets an absolute deadline.
func (t *Task) SetDeadline(at time.Time) error {
	if t == nil {
		return fmt.Errorf("%w: task is nil", contracts.ErrInvalidArgument)
	}
	if at.IsZero() {
		return fmt.Errorf("%w: deadline required", contracts.ErrInvalidArgument)
	}
	ts := at.UTC()
	t.Deadline = &ts
	t.UpdatedAt = contracts.NowUTC()
	return nil
}

// IsPastDeadline reports whether now is after the task deadline.
func (t *Task) IsPastDeadline(now time.Time) bool {
	if t == nil || t.Deadline == nil {
		return false
	}
	return now.After(*t.Deadline)
}

// CanRetry reports whether another attempt is allowed by policy.
func (t *Task) CanRetry() bool {
	if t == nil || t.Retry == nil {
		return false
	}
	return t.Retry.Attempts < t.Retry.Policy.MaxAttempts
}

// RecordAttempt increments attempt counter after a failed execution.
func (t *Task) RecordAttempt(errMsg string, now time.Time) error {
	if t == nil {
		return fmt.Errorf("%w: task is nil", contracts.ErrInvalidArgument)
	}
	if t.Retry == nil {
		return fmt.Errorf("%w: no retry policy", contracts.ErrInvalidArgument)
	}
	t.Retry.Attempts++
	t.Retry.LastError = errMsg
	if t.CanRetry() && t.Retry.Policy.Backoff > 0 {
		n := now.Add(t.Retry.Policy.Backoff)
		t.Retry.NextRetry = &n
	} else {
		t.Retry.NextRetry = nil
	}
	t.UpdatedAt = contracts.NowUTC()
	return nil
}

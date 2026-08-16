// Package taskengine implements the Task Engine: creation and explicit
// state transitions for tasks bound to Project / Scope / Principal / Capability.
//
// Illegal transitions are rejected. Authorization of work remains the
// responsibility of the Control Plane (Phase 1E). This package does not
// issue ExecutionGrants or invoke the Tool Broker.
package taskengine

import (
	"context"
	"fmt"
	"time"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/capability"
	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
	"cyberos/internal/identity"
	"cyberos/internal/storage/ports"
)

// Engine manages task lifecycle.
type Engine struct {
	tasks    ports.TaskRepository
	identity *identity.Service
	scopes   *appscope.Service
	caps     *capability.Registry
}

// NewEngine constructs a Task Engine.
// tasks repo may be nil for pure in-memory unit tests that only exercise
// transition logic via CreateInMemory helpers; production wiring must supply repos.
func NewEngine(
	tasks ports.TaskRepository,
	idSvc *identity.Service,
	scopeSvc *appscope.Service,
	capReg *capability.Registry,
) *Engine {
	return &Engine{
		tasks:    tasks,
		identity: idSvc,
		scopes:   scopeSvc,
		caps:     capReg,
	}
}

// CreateRequest carries the data needed to create a task.
type CreateRequest struct {
	ID           contracts.TaskID
	ProjectID    contracts.ProjectID
	ScopeID      contracts.ScopeID
	PrincipalID  contracts.PrincipalID
	CapabilityID contracts.CapabilityID
	Type         contracts.TaskType
	Title        string
	Description  string
	// Now overrides the clock for scope validity checks.
	Now time.Time
}

// Create validates bindings (principal active, scope active, capability known)
// and persists a task in CREATED state.
//
// Capability presence is discovery only — Create does NOT authorize execution.
func (e *Engine) Create(ctx context.Context, req CreateRequest) (*task.Task, error) {
	if e == nil {
		return nil, fmt.Errorf("%w: task engine not configured", contracts.ErrInvalidArgument)
	}
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	// Principal must be active
	if e.identity != nil {
		if _, err := e.identity.RequireActive(ctx, req.PrincipalID); err != nil {
			return nil, fmt.Errorf("principal: %w", err)
		}
	}

	// Scope must be active and belong to project
	if e.scopes != nil {
		sc, err := e.scopes.RequireActive(ctx, req.ScopeID, now)
		if err != nil {
			return nil, fmt.Errorf("scope: %w", err)
		}
		if err := appscope.EnsureSameProject(sc, req.ProjectID); err != nil {
			return nil, err
		}
	}

	// Capability must be known (discovery only — not authorization)
	if e.caps != nil {
		if _, err := e.caps.RequireKnown(ctx, req.CapabilityID); err != nil {
			return nil, fmt.Errorf("capability: %w", err)
		}
	}

	t, err := task.New(req.ID, req.ProjectID, req.ScopeID, req.PrincipalID, req.CapabilityID, req.Type, req.Title)
	if err != nil {
		return nil, err
	}
	t.Description = req.Description

	if e.tasks != nil {
		if err := e.tasks.Create(ctx, t); err != nil {
			return nil, err
		}
	}
	return t, nil
}

// Get loads a task by ID.
func (e *Engine) Get(ctx context.Context, id contracts.TaskID) (*task.Task, error) {
	if e == nil || e.tasks == nil {
		return nil, fmt.Errorf("%w: task repository not configured", contracts.ErrInvalidArgument)
	}
	return e.tasks.GetByID(ctx, id)
}

// Transition moves a task to a new state if the transition is legal.
// Illegal transitions return ErrInvalidStateTransition and leave state unchanged.
func (e *Engine) Transition(ctx context.Context, id contracts.TaskID, next contracts.TaskState) (*task.Task, error) {
	if e == nil || e.tasks == nil {
		return nil, fmt.Errorf("%w: task repository not configured", contracts.ErrInvalidArgument)
	}
	t, err := e.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := t.TransitionTo(next); err != nil {
		return nil, err
	}
	if err := e.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ListByScope returns tasks for a scope.
func (e *Engine) ListByScope(ctx context.Context, scopeID contracts.ScopeID, limit, offset int) ([]*task.Task, error) {
	if e == nil || e.tasks == nil {
		return nil, fmt.Errorf("%w: task repository not configured", contracts.ErrInvalidArgument)
	}
	return e.tasks.ListByScope(ctx, scopeID, limit, offset)
}

// ScheduleRequest schedules an existing task without authorizing it.
type ScheduleRequest struct {
	TaskID    contracts.TaskID
	NotBefore time.Time
	Priority  int
}

// Schedule applies schedule metadata via domain helpers and transitions
// CREATED → SCHEDULED when appropriate. It does not call Control Plane.
func (e *Engine) Schedule(ctx context.Context, req ScheduleRequest) (*task.Task, error) {
	if e == nil || e.tasks == nil {
		return nil, fmt.Errorf("%w: task repository not configured", contracts.ErrInvalidArgument)
	}
	t, err := e.tasks.GetByID(ctx, req.TaskID)
	if err != nil {
		return nil, err
	}
	if err := t.EnsureBindings(); err != nil {
		return nil, err
	}
	if err := t.ApplySchedule(task.ScheduleSpec{
		NotBefore: req.NotBefore,
		Priority:  req.Priority,
	}); err != nil {
		return nil, err
	}
	switch t.State {
	case contracts.TaskStateCreated:
		if err := t.TransitionTo(contracts.TaskStateScheduled); err != nil {
			return nil, err
		}
	case contracts.TaskStateScheduled:
		// allow reschedule
	default:
		return nil, fmt.Errorf("%w: cannot schedule task in state %s",
			contracts.ErrInvalidStateTransition, t.State)
	}
	if err := e.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// PromoteScheduled moves a due SCHEDULED task to PENDING.
// This is a lifecycle step only — not authorization.
func (e *Engine) PromoteScheduled(ctx context.Context, id contracts.TaskID, now time.Time) (*task.Task, error) {
	if e == nil || e.tasks == nil {
		return nil, fmt.Errorf("%w: task repository not configured", contracts.ErrInvalidArgument)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	t, err := e.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t.State != contracts.TaskStateScheduled {
		return nil, fmt.Errorf("%w: task is not SCHEDULED", contracts.ErrInvalidStateTransition)
	}
	if !t.IsDue(now) {
		return nil, fmt.Errorf("%w: task is not due yet", contracts.ErrInvalidArgument)
	}
	if err := t.TransitionTo(contracts.TaskStatePending); err != nil {
		return nil, err
	}
	if t.Schedule != nil {
		q := now
		t.Schedule.QueuedAt = &q
	}
	if err := e.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// SetDependencies attaches dependency IDs after graph validation.
// depsLookup is used for existence and cycle checks; may be nil to skip graph validation
// (not recommended). Prefer wiring a dependency.Graph via ValidateDependencies first.
func (e *Engine) SetDependencies(ctx context.Context, id contracts.TaskID, dependsOn []contracts.TaskID) (*task.Task, error) {
	if e == nil || e.tasks == nil {
		return nil, fmt.Errorf("%w: task repository not configured", contracts.ErrInvalidArgument)
	}
	t, err := e.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := t.SetDependencies(dependsOn); err != nil {
		return nil, err
	}
	if err := e.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ApplyRetryPolicy sets a bounded retry policy on the task.
func (e *Engine) ApplyRetryPolicy(ctx context.Context, id contracts.TaskID, maxAttempts int, backoff time.Duration) (*task.Task, error) {
	if e == nil || e.tasks == nil {
		return nil, fmt.Errorf("%w: task repository not configured", contracts.ErrInvalidArgument)
	}
	t, err := e.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := t.ApplyRetryPolicy(task.RetryPolicy{MaxAttempts: maxAttempts, Backoff: backoff}); err != nil {
		return nil, err
	}
	if err := e.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// SetTimeoutDeadline sets timeout and/or absolute deadline.
func (e *Engine) SetTimeoutDeadline(ctx context.Context, id contracts.TaskID, timeout time.Duration, deadline *time.Time) (*task.Task, error) {
	if e == nil || e.tasks == nil {
		return nil, fmt.Errorf("%w: task repository not configured", contracts.ErrInvalidArgument)
	}
	t, err := e.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if timeout != 0 {
		if err := t.SetTimeout(timeout); err != nil {
			return nil, err
		}
	}
	if deadline != nil {
		if err := t.SetDeadline(*deadline); err != nil {
			return nil, err
		}
	}
	if err := e.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// MarkFailed records a failed attempt. If retries remain, transitions FAILED→PENDING
// (after optional backoff window is left to the caller/scheduler).
// Does not issue grants or execute.
func (e *Engine) MarkFailed(ctx context.Context, id contracts.TaskID, errMsg string, now time.Time) (*task.Task, error) {
	if e == nil || e.tasks == nil {
		return nil, fmt.Errorf("%w: task repository not configured", contracts.ErrInvalidArgument)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	t, err := e.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Transition to FAILED if currently RUNNING
	if t.State == contracts.TaskStateRunning {
		if err := t.TransitionTo(contracts.TaskStateFailed); err != nil {
			return nil, err
		}
	}
	if t.Retry != nil {
		if err := t.RecordAttempt(errMsg, now); err != nil {
			return nil, err
		}
		if t.CanRetry() {
			// Move back to PENDING for another authorization+execution cycle later.
			if err := t.TransitionTo(contracts.TaskStatePending); err != nil {
				return nil, err
			}
		}
	}
	if err := e.tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// AssertNotTimedOut returns an error if the task is past its deadline.
func (e *Engine) AssertNotTimedOut(ctx context.Context, id contracts.TaskID, now time.Time) error {
	if e == nil || e.tasks == nil {
		return fmt.Errorf("%w: task repository not configured", contracts.ErrInvalidArgument)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	t, err := e.tasks.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if t.IsPastDeadline(now) {
		return fmt.Errorf("%w: task deadline exceeded", contracts.ErrInvalidArgument)
	}
	return nil
}

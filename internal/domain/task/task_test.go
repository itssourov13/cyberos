package task

import (
	"errors"
	"testing"
	"time"

	"cyberos/internal/contracts"
)

func newTestTask(t *testing.T) *Task {
	t.Helper()
	task, err := New(
		"task-1", "proj-1", "scope-1", "user-1", "cap-1",
		contracts.TaskTypeRecon, "Enumerate domains",
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return task
}

func TestNewStartsInCreated(t *testing.T) {
	task := newTestTask(t)
	if task.State != contracts.TaskStateCreated {
		t.Errorf("expected CREATED, got %s", task.State)
	}
	if err := task.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
}

func TestValidTransitions(t *testing.T) {
	task := newTestTask(t)

	// CREATED → PENDING
	if err := task.TransitionTo(contracts.TaskStatePending); err != nil {
		t.Fatalf("CREATED→PENDING: %v", err)
	}
	// PENDING → AUTHORIZED
	if err := task.TransitionTo(contracts.TaskStateAuthorized); err != nil {
		t.Fatalf("PENDING→AUTHORIZED: %v", err)
	}
	// AUTHORIZED → READY
	if err := task.TransitionTo(contracts.TaskStateReady); err != nil {
		t.Fatalf("AUTHORIZED→READY: %v", err)
	}
	// READY → RUNNING
	if err := task.TransitionTo(contracts.TaskStateRunning); err != nil {
		t.Fatalf("READY→RUNNING: %v", err)
	}
	// RUNNING → SUCCEEDED
	if err := task.TransitionTo(contracts.TaskStateSucceeded); err != nil {
		t.Fatalf("RUNNING→SUCCEEDED: %v", err)
	}
	if !task.IsTerminal() {
		t.Error("SUCCEEDED should be terminal")
	}
}

func TestIllegalTransition(t *testing.T) {
	task := newTestTask(t)
	// CREATED → RUNNING is illegal
	err := task.TransitionTo(contracts.TaskStateRunning)
	if err == nil {
		t.Fatal("CREATED→RUNNING should be rejected")
	}
	if !errors.Is(err, contracts.ErrInvalidStateTransition) {
		t.Errorf("expected ErrInvalidStateTransition, got %v", err)
	}
	// State must remain unchanged
	if task.State != contracts.TaskStateCreated {
		t.Errorf("state mutated on illegal transition: %s", task.State)
	}
}

func TestTerminalHasNoOutbound(t *testing.T) {
	task := newTestTask(t)
	_ = task.TransitionTo(contracts.TaskStatePending)
	_ = task.TransitionTo(contracts.TaskStateAuthorized)
	_ = task.TransitionTo(contracts.TaskStateReady)
	_ = task.TransitionTo(contracts.TaskStateRunning)
	_ = task.TransitionTo(contracts.TaskStateSucceeded)

	if err := task.TransitionTo(contracts.TaskStateRunning); err == nil {
		t.Error("terminal state must reject further transitions")
	}
}

func TestCancelFromCreated(t *testing.T) {
	task := newTestTask(t)
	if err := task.TransitionTo(contracts.TaskStateCancelled); err != nil {
		t.Fatalf("CREATED→CANCELLED should be allowed: %v", err)
	}
	if !task.IsTerminal() {
		t.Error("CANCELLED should be terminal")
	}
}

func TestMissingFields(t *testing.T) {
	if _, err := New("", "p", "s", "u", "c", contracts.TaskTypeRecon, "t"); err == nil {
		t.Error("empty id should fail")
	}
	if _, err := New("id", "p", "s", "u", "c", contracts.TaskTypeRecon, ""); err == nil {
		t.Error("empty title should fail")
	}
}

func TestScheduledTransition(t *testing.T) {
	task := newTestTask(t)
	if err := task.TransitionTo(contracts.TaskStateScheduled); err != nil {
		t.Fatalf("CREATED→SCHEDULED: %v", err)
	}
	if err := task.TransitionTo(contracts.TaskStatePending); err != nil {
		t.Fatalf("SCHEDULED→PENDING: %v", err)
	}
}

func TestScheduleDoesNotSkipToRunning(t *testing.T) {
	task := newTestTask(t)
	_ = task.TransitionTo(contracts.TaskStateScheduled)
	err := task.TransitionTo(contracts.TaskStateRunning)
	if err == nil {
		t.Fatal("SCHEDULED→RUNNING must be illegal")
	}
}

func TestApplyScheduleAndIsDue(t *testing.T) {
	task := newTestTask(t)
	now := time.Now().UTC()
	if err := task.ApplySchedule(ScheduleSpec{NotBefore: now.Add(-time.Minute)}); err != nil {
		t.Fatal(err)
	}
	_ = task.TransitionTo(contracts.TaskStateScheduled)
	if !task.IsDue(now) {
		t.Error("should be due")
	}
	if task.IsDue(now.Add(-time.Hour)) {
		t.Error("past clock before not_before should not be due")
	}
}

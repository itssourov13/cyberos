package scheduler

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
)

func seedTask(store *MemoryStore, id string) *task.Task {
	t, _ := task.New(
		contracts.TaskID(id), "proj-1", "scope-1", "user-1", "cap-1",
		contracts.TaskTypeRecon, "T",
	)
	store.Put(t)
	return t
}

func TestScheduleAndTickPromotesDue(t *testing.T) {
	store := NewMemoryStore()
	seedTask(store, "task-1")
	sch := New(store)
	fixed := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	sch.Now = func() time.Time { return fixed }

	tk, err := sch.Schedule(ScheduleRequest{
		TaskID: "task-1", NotBefore: fixed.Add(-time.Minute), Priority: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if tk.State != contracts.TaskStateScheduled {
		t.Fatalf("state = %s", tk.State)
	}

	promoted, err := sch.Tick()
	if err != nil {
		t.Fatal(err)
	}
	if len(promoted) != 1 {
		t.Fatalf("promoted = %d", len(promoted))
	}
	if promoted[0].State != contracts.TaskStatePending {
		t.Fatalf("expected PENDING, got %s", promoted[0].State)
	}
	if promoted[0].Schedule == nil || promoted[0].Schedule.QueuedAt == nil {
		t.Fatal("queued_at should be set")
	}
}

func TestTickDoesNotPromoteFuture(t *testing.T) {
	store := NewMemoryStore()
	seedTask(store, "task-2")
	sch := New(store)
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	sch.Now = func() time.Time { return now }

	_, err := sch.Schedule(ScheduleRequest{
		TaskID: "task-2", NotBefore: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	promoted, err := sch.Tick()
	if err != nil {
		t.Fatal(err)
	}
	if len(promoted) != 0 {
		t.Fatalf("future task must not promote, got %d", len(promoted))
	}
	got, _ := store.GetByID("task-2")
	if got.State != contracts.TaskStateScheduled {
		t.Fatalf("state = %s", got.State)
	}
}

func TestScheduleDoesNotAuthorize(t *testing.T) {
	// Invariant check: after schedule+tick, state is PENDING not AUTHORIZED/READY/RUNNING.
	store := NewMemoryStore()
	seedTask(store, "task-3")
	sch := New(store)
	now := time.Now().UTC()
	sch.Now = func() time.Time { return now }
	_, _ = sch.Schedule(ScheduleRequest{TaskID: "task-3", NotBefore: now.Add(-time.Second)})
	promoted, _ := sch.Tick()
	if len(promoted) != 1 {
		t.Fatal("expected promotion")
	}
	if promoted[0].State == contracts.TaskStateAuthorized ||
		promoted[0].State == contracts.TaskStateReady ||
		promoted[0].State == contracts.TaskStateRunning {
		t.Fatal("scheduler must not authorize or start execution")
	}
	if promoted[0].State != contracts.TaskStatePending {
		t.Fatalf("expected PENDING, got %s", promoted[0].State)
	}
}

func TestCannotScheduleFromRunning(t *testing.T) {
	store := NewMemoryStore()
	tk := seedTask(store, "task-4")
	_ = tk.TransitionTo(contracts.TaskStatePending)
	_ = tk.TransitionTo(contracts.TaskStateAuthorized)
	_ = tk.TransitionTo(contracts.TaskStateReady)
	_ = tk.TransitionTo(contracts.TaskStateRunning)
	store.Put(tk)

	sch := New(store)
	_, err := sch.Schedule(ScheduleRequest{
		TaskID: "task-4", NotBefore: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("scheduling running task must fail")
	}
}

func TestPriorityOrdering(t *testing.T) {
	store := NewMemoryStore()
	seedTask(store, "low")
	seedTask(store, "high")
	sch := New(store)
	now := time.Now().UTC()
	sch.Now = func() time.Time { return now }
	_, _ = sch.Schedule(ScheduleRequest{TaskID: "low", NotBefore: now.Add(-time.Minute), Priority: 1})
	_, _ = sch.Schedule(ScheduleRequest{TaskID: "high", NotBefore: now.Add(-time.Minute), Priority: 100})
	promoted, err := sch.Tick()
	if err != nil {
		t.Fatal(err)
	}
	if len(promoted) != 2 {
		t.Fatalf("got %d", len(promoted))
	}
	if promoted[0].ID != "high" {
		t.Errorf("expected high priority first, got %s", promoted[0].ID)
	}
}

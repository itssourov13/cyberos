package runner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
	"cyberos/internal/events"
)

func TestWorkerTickOnceClaimRelease(t *testing.T) {
	store := NewMemoryStore()
	tk, _ := task.New("t1", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	tk.State = contracts.TaskStatePending
	store.Put(tk)

	r := New(store, time.Minute)
	es := events.NewStore()
	var processed atomic.Int32
	w := NewWorker(r, "w1", func(ctx context.Context, c Claim) error {
		processed.Add(1)
		if c.Task.State != contracts.TaskStatePending {
			t.Errorf("process saw state %s", c.Task.State)
		}
		return nil
	})
	w.Events = es

	n, err := w.TickOnce(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("tick: n=%d err=%v", n, err)
	}
	if processed.Load() != 1 {
		t.Fatal("process not called")
	}
	if len(es.ByType(events.TypeRunnerClaimed)) < 1 {
		t.Fatal("expected claimed event")
	}
	if len(es.ByType(events.TypeRunnerReleased)) < 1 {
		t.Fatal("expected released event")
	}
	if len(r.ActiveLeases()) != 0 {
		t.Fatal("lease should be released")
	}
}

func TestWorkerDoesNotExecuteWithoutHandler(t *testing.T) {
	store := NewMemoryStore()
	tk, _ := task.New("t2", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	tk.State = contracts.TaskStatePending
	store.Put(tk)
	w := NewWorker(New(store, time.Minute), "w2", nil)
	n, err := w.TickOnce(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("tick: %v %d", err, n)
	}
	// State still PENDING — no auth/execution
	got, _ := store.GetByID(context.Background(), "t2")
	if got.State != contracts.TaskStatePending {
		t.Fatalf("state=%s", got.State)
	}
}

func TestWorkerPromoteEmitsEvent(t *testing.T) {
	store := NewMemoryStore()
	tk, _ := task.New("t3", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	now := time.Now().UTC()
	_ = tk.ApplySchedule(task.ScheduleSpec{NotBefore: now.Add(-time.Second)})
	_ = tk.TransitionTo(contracts.TaskStateScheduled)
	store.Put(tk)
	r := New(store, time.Minute)
	r.Now = func() time.Time { return now }
	es := events.NewStore()
	w := NewWorker(r, "w3", nil)
	w.Events = es
	_, _ = w.TickOnce(context.Background())
	if len(es.ByType(events.TypeRunnerPromoted)) < 1 {
		t.Fatal("expected promoted event")
	}
}

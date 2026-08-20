package runner

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
)

func seed(store *MemoryStore, id string, state contracts.TaskState) *task.Task {
	t, _ := task.New(contracts.TaskID(id), "p", "s", "u", "c", contracts.TaskTypeCustom, id)
	t.State = state
	store.Put(t)
	return t
}

func TestPromoteDueScheduled(t *testing.T) {
	store := NewMemoryStore()
	tk := seed(store, "t1", contracts.TaskStateCreated)
	now := time.Now().UTC()
	_ = tk.ApplySchedule(task.ScheduleSpec{NotBefore: now.Add(-time.Minute)})
	_ = tk.TransitionTo(contracts.TaskStateScheduled)
	store.Put(tk)

	r := New(store, time.Second)
	r.Now = func() time.Time { return now }
	out, err := r.PromoteDueScheduled(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].State != contracts.TaskStatePending {
		t.Fatalf("promote: %+v", out)
	}
}

func TestClaimAndRecoverLease(t *testing.T) {
	store := NewMemoryStore()
	seed(store, "t2", contracts.TaskStatePending)
	r := New(store, 50*time.Millisecond)
	claims, err := r.ClaimPending(context.Background(), "worker-1", 5)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim: %v %d", err, len(claims))
	}
	// Second claim should not re-claim while lease active
	claims2, _ := r.ClaimPending(context.Background(), "worker-2", 5)
	if len(claims2) != 0 {
		t.Fatal("should not double-claim")
	}
	time.Sleep(60 * time.Millisecond)
	n := r.RecoverExpiredLeases()
	if n != 1 {
		t.Fatalf("recovered=%d", n)
	}
	claims3, _ := r.ClaimPending(context.Background(), "worker-2", 5)
	if len(claims3) != 1 {
		t.Fatal("should reclaim after recovery")
	}
	r.Release("t2")
}

func TestClaimDoesNotAuthorize(t *testing.T) {
	store := NewMemoryStore()
	seed(store, "t3", contracts.TaskStatePending)
	r := New(store, time.Minute)
	claims, _ := r.ClaimPending(context.Background(), "w", 1)
	if claims[0].Task.State != contracts.TaskStatePending {
		t.Fatal("claim must leave state PENDING (not AUTHORIZED/RUNNING)")
	}
}

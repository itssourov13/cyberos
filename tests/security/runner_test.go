package security_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
	"cyberos/internal/runner"
)

// Runner claim/promote never moves a task into authorized or running state.
func TestSecurity_RunnerDoesNotAuthorizeOrExecute(t *testing.T) {
	store := runner.NewMemoryStore()
	tk, _ := task.New("t1", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	now := time.Now().UTC()
	_ = tk.ApplySchedule(task.ScheduleSpec{NotBefore: now.Add(-time.Second)})
	_ = tk.TransitionTo(contracts.TaskStateScheduled)
	store.Put(tk)

	r := runner.New(store, time.Minute)
	r.Now = func() time.Time { return now }
	promoted, err := r.PromoteDueScheduled(context.Background(), 10)
	if err != nil || len(promoted) != 1 {
		t.Fatalf("promote: %v", err)
	}
	claims, err := r.ClaimPending(context.Background(), "worker", 10)
	if err != nil || len(claims) != 1 {
		t.Fatalf("claim: %v", err)
	}
	st := claims[0].Task.State
	for _, bad := range []contracts.TaskState{
		contracts.TaskStateAuthorized, contracts.TaskStateReady,
		contracts.TaskStateRunning, contracts.TaskStateSucceeded,
	} {
		if st == bad {
			t.Fatalf("runner reached forbidden state %s", bad)
		}
	}
}

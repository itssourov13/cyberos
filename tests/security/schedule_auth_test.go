package security_test

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
	"cyberos/internal/scheduler"
)

// Security invariant: scheduling a task never places it in an authorized
// or executable state. Control Plane remains mandatory.
func TestSecurity_ScheduleIsNotAuthorization(t *testing.T) {
	store := scheduler.NewMemoryStore()
	tk, _ := task.New("t1", "p1", "s1", "u1", "c1", contracts.TaskTypeCustom, "x")
	store.Put(tk)

	sch := scheduler.New(store)
	now := time.Now().UTC()
	sch.Now = func() time.Time { return now }

	scheduled, err := sch.Schedule(scheduler.ScheduleRequest{
		TaskID: "t1", NotBefore: now.Add(-time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if scheduled.State != contracts.TaskStateScheduled {
		t.Fatalf("state=%s", scheduled.State)
	}

	promoted, err := sch.Tick()
	if err != nil {
		t.Fatal(err)
	}
	if len(promoted) != 1 || promoted[0].State != contracts.TaskStatePending {
		t.Fatalf("expected PENDING after tick, got %+v", promoted)
	}

	// Explicitly forbidden outcomes after schedule-only path:
	forbidden := []contracts.TaskState{
		contracts.TaskStateAuthorized,
		contracts.TaskStateReady,
		contracts.TaskStateRunning,
		contracts.TaskStateSucceeded,
	}
	for _, st := range forbidden {
		if promoted[0].State == st {
			t.Fatalf("schedule path reached forbidden state %s", st)
		}
	}
}

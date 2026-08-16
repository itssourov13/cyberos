package task

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
)

func TestDependenciesAndRetryPolicy(t *testing.T) {
	tk := newTestTask(t)
	if err := tk.SetDependencies([]contracts.TaskID{"other"}); err != nil {
		t.Fatal(err)
	}
	if err := tk.SetDependencies([]contracts.TaskID{tk.ID}); err == nil {
		t.Fatal("self dep must fail")
	}
	if err := tk.ApplyRetryPolicy(RetryPolicy{MaxAttempts: 3, Backoff: time.Second}); err != nil {
		t.Fatal(err)
	}
	if tk.CanRetry() != true {
		t.Error("should allow retry before attempts")
	}
	_ = tk.RecordAttempt("boom", time.Now().UTC())
	if tk.Retry.Attempts != 1 {
		t.Errorf("attempts=%d", tk.Retry.Attempts)
	}
	_ = tk.RecordAttempt("boom2", time.Now().UTC())
	_ = tk.RecordAttempt("boom3", time.Now().UTC())
	if tk.CanRetry() {
		t.Error("max attempts exhausted")
	}
}

func TestRetryHardLimit(t *testing.T) {
	tk := newTestTask(t)
	if err := tk.ApplyRetryPolicy(RetryPolicy{MaxAttempts: 101}); err == nil {
		t.Fatal("hard limit 100")
	}
}

func TestTimeoutAndDeadline(t *testing.T) {
	tk := newTestTask(t)
	if err := tk.SetTimeout(-1); err == nil {
		t.Fatal("negative timeout")
	}
	if err := tk.SetTimeout(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	dl := time.Now().UTC().Add(-time.Minute)
	_ = tk.SetDeadline(dl)
	if !tk.IsPastDeadline(time.Now().UTC()) {
		t.Error("should be past deadline")
	}
}

func TestFailedCanTransitionToPendingForRetry(t *testing.T) {
	tk := newTestTask(t)
	_ = tk.TransitionTo(contracts.TaskStatePending)
	_ = tk.TransitionTo(contracts.TaskStateAuthorized)
	_ = tk.TransitionTo(contracts.TaskStateReady)
	_ = tk.TransitionTo(contracts.TaskStateRunning)
	_ = tk.TransitionTo(contracts.TaskStateFailed)
	if err := tk.TransitionTo(contracts.TaskStatePending); err != nil {
		t.Fatalf("FAILED→PENDING for retry: %v", err)
	}
}

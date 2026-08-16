package security_test

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/dependency"
	"cyberos/internal/domain/task"
)

type memL struct {
	items map[contracts.TaskID]*task.Task
}

func (m *memL) GetByID(id contracts.TaskID) (*task.Task, error) {
	t, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *t
	return &cp, nil
}

// Dependencies satisfied ≠ authorized to run.
func TestSecurity_DependenciesDoNotAuthorize(t *testing.T) {
	m := &memL{items: map[contracts.TaskID]*task.Task{}}
	a, _ := task.New("a", "p", "s", "u", "c", contracts.TaskTypeCustom, "A")
	a.State = contracts.TaskStateSucceeded
	b, _ := task.New("b", "p", "s", "u", "c", contracts.TaskTypeCustom, "B")
	b.DependsOn = []contracts.TaskID{"a"}
	b.State = contracts.TaskStatePending
	m.items["a"] = a
	m.items["b"] = b

	g := dependency.New(m)
	if err := g.AssertRunnable(b); err != nil {
		t.Fatalf("deps should be satisfied: %v", err)
	}
	// Still PENDING — not AUTHORIZED/READY/RUNNING
	if b.State != contracts.TaskStatePending {
		t.Fatalf("state=%s", b.State)
	}
}

// Infinite retries are hard-capped.
func TestSecurity_RetryHardCap(t *testing.T) {
	tk, _ := task.New("t", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	if err := tk.ApplyRetryPolicy(task.RetryPolicy{MaxAttempts: 1000}); err == nil {
		t.Fatal("must reject unbounded retries")
	}
	_ = tk.ApplyRetryPolicy(task.RetryPolicy{MaxAttempts: 2})
	now := time.Now().UTC()
	_ = tk.RecordAttempt("1", now)
	_ = tk.RecordAttempt("2", now)
	if tk.CanRetry() {
		t.Fatal("attempts exhausted")
	}
}

// Past deadline blocks further work at policy layer.
func TestSecurity_DeadlineBlocks(t *testing.T) {
	tk, _ := task.New("t", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	past := time.Now().UTC().Add(-time.Hour)
	_ = tk.SetDeadline(past)
	if !tk.IsPastDeadline(time.Now().UTC()) {
		t.Fatal("deadline should block")
	}
}

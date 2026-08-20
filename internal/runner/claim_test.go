package runner

import (
	"context"
	"sync"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
)

func TestConcurrentClaimOnlyOneWins(t *testing.T) {
	store := NewMemoryStore()
	tk, _ := task.New("t1", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	_ = tk.TransitionTo(contracts.TaskStatePending)
	store.Put(tk)

	r := New(store, time.Minute)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var wins int
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			claims, err := r.ClaimPending(context.Background(), WorkerID(id), 1)
			if err != nil {
				t.Error(err)
				return
			}
			if len(claims) == 1 {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}(string(rune('a' + i)))
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("expected exactly 1 successful claim, got %d", wins)
	}
}

func TestLeaseRecoveryAllowsReclaim(t *testing.T) {
	store := NewMemoryStore()
	tk, _ := task.New("t2", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	_ = tk.TransitionTo(contracts.TaskStatePending)
	store.Put(tk)

	now := time.Now().UTC()
	r := New(store, time.Second)
	r.Now = func() time.Time { return now }

	claims, _ := r.ClaimPending(context.Background(), "w1", 1)
	if len(claims) != 1 {
		t.Fatal(claims)
	}
	// advance past lease
	r.Now = func() time.Time { return now.Add(2 * time.Second) }
	n := r.RecoverExpiredLeases()
	if n < 1 {
		t.Fatalf("recovered=%d", n)
	}
	claims2, _ := r.ClaimPending(context.Background(), "w2", 1)
	if len(claims2) != 1 {
		t.Fatalf("reclaim failed: %v", claims2)
	}
	if claims2[0].WorkerID != "w2" {
		t.Fatal(claims2[0].WorkerID)
	}
}

func TestTerminalTaskNotClaimable(t *testing.T) {
	store := NewMemoryStore()
	tk, _ := task.New("t3", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	tk.State = contracts.TaskStateSucceeded
	store.Put(tk)
	r := New(store, time.Minute)
	claims, _ := r.ClaimPending(context.Background(), "w", 5)
	if len(claims) != 0 {
		t.Fatal("terminal must not be claimed")
	}
}

func TestClaimDoesNotChangeToAuthorized(t *testing.T) {
	store := NewMemoryStore()
	tk, _ := task.New("t4", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	_ = tk.TransitionTo(contracts.TaskStatePending)
	store.Put(tk)
	r := New(store, time.Minute)
	claims, _ := r.ClaimPending(context.Background(), "w", 1)
	if len(claims) != 1 {
		t.Fatal(claims)
	}
	if claims[0].Task.State != contracts.TaskStatePending {
		t.Fatalf("claim must not authorize: %s", claims[0].Task.State)
	}
}

package security_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
	"cyberos/internal/runner"
)

func TestSecurity_ClaimIsNotAuthorization(t *testing.T) {
	store := runner.NewMemoryStore()
	tk, _ := task.New("tx", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	_ = tk.TransitionTo(contracts.TaskStatePending)
	store.Put(tk)
	r := runner.New(store, time.Minute)
	claims, err := r.ClaimPending(context.Background(), "worker-a", 1)
	if err != nil || len(claims) != 1 {
		t.Fatalf("%v %v", err, claims)
	}
	// Still PENDING — no Control Plane, no grant, no broker.
	if claims[0].Task.State != contracts.TaskStatePending {
		t.Fatal(claims[0].Task.State)
	}
}

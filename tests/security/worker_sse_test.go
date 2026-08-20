package security_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/task"
	"cyberos/internal/events"
	"cyberos/internal/runner"
)

// Worker process without grant path must not flip task to RUNNING.
func TestSecurity_WorkerLoopNotAuthorization(t *testing.T) {
	store := runner.NewMemoryStore()
	tk, _ := task.New("t1", "p", "s", "u", "c", contracts.TaskTypeCustom, "T")
	tk.State = contracts.TaskStatePending
	store.Put(tk)

	executed := false
	w := runner.NewWorker(runner.New(store, time.Minute), "w", func(ctx context.Context, c runner.Claim) error {
		executed = true
		// Simulate forbidden: worker must NOT transition to RUNNING here without CP/grant.
		return nil
	})
	w.Events = events.NewStore()
	_, _ = w.TickOnce(context.Background())
	if !executed {
		t.Fatal("handler should run")
	}
	got, _ := store.GetByID(context.Background(), "t1")
	if got.State == contracts.TaskStateRunning || got.State == contracts.TaskStateAuthorized {
		t.Fatalf("worker must not authorize/run: state=%s", got.State)
	}
}

// SSE hub is observation only — publishing does not change task state.
func TestSecurity_SSEIsReadOnly(t *testing.T) {
	hub := events.NewHub()
	ch := hub.Subscribe(1)
	hub.Publish(contracts.EventEnvelope{
		ID: "x", Type: events.TypeRunnerClaimed, SchemaVersion: contracts.EventSchemaV1,
		Timestamp: time.Now().UTC(), CorrelationID: "c",
	})
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
	// No task store involved — pure fan-out.
}

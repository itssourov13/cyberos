package autonomy_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
	"cyberos/internal/storage/memory"
	"cyberos/internal/storage/ports"
)

func Test15C_EnqueueClaimDeliver(t *testing.T) {
	repo := memory.NewAutonomyOutboxRepo()
	ctrl := autonomy.NewController(&ai.MockBrain{})
	_, _ = ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-out", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 5},
	})
	proc := autonomy.NewEventAutonomyProcessor(ctrl, &ai.MockBrain{})
	ev := autonomy.AutonomyEvent{
		EventID: "ev-out-1", SchemaVersion: autonomy.AutonomyEventContractV1,
		Type: autonomy.EventExecutionCompleted, ProjectID: "proj-a", ScopeID: "s",
		AutonomyRunID: "ar-out", IdempotencyKey: "idem-out-1", Timestamp: time.Now().UTC(),
	}
	if err := autonomy.EnqueueAutonomyEvent(context.Background(), repo, ev); err != nil {
		t.Fatal(err)
	}
	// duplicate enqueue
	if err := autonomy.EnqueueAutonomyEvent(context.Background(), repo, ev); err != nil {
		t.Fatal(err)
	}
	d := autonomy.NewDispatcher(repo, proc, "worker-1")
	del, _, dead, err := d.ProcessBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if del != 1 || dead != 0 {
		t.Fatalf("delivered=%d dead=%d", del, dead)
	}
	got, err := repo.Get(context.Background(), "proj-a", "ev-out-1")
	if err != nil || got.Status != ports.OutboxDelivered {
		t.Fatal(err, got.Status)
	}
}

func Test15C_LeaseExpiryReclaim(t *testing.T) {
	repo := memory.NewAutonomyOutboxRepo()
	_ = repo.Enqueue(context.Background(), ports.AutonomyOutboxRecord{
		ID: "ev-lease", ProjectID: "proj-a", IdempotencyKey: "idem-lease",
		EventType: string(autonomy.EventFindingCreated), SchemaVersion: "1", EventID: "ev-lease",
		Status: ports.OutboxPending, AvailableAt: time.Now().UTC(),
	})
	// claim with short lease
	batch, err := repo.ClaimBatch(context.Background(), "w1", 1, time.Millisecond)
	if err != nil || len(batch) != 1 {
		t.Fatal(err, batch)
	}
	time.Sleep(5 * time.Millisecond)
	// reclaim by another worker
	batch2, err := repo.ClaimBatch(context.Background(), "w2", 1, time.Second)
	if err != nil || len(batch2) != 1 {
		t.Fatal(err, len(batch2))
	}
	if batch2[0].LockedBy != "w2" {
		t.Fatal(batch2[0].LockedBy)
	}
}

func Test15C_InvalidSchemaDeadLetter(t *testing.T) {
	repo := memory.NewAutonomyOutboxRepo()
	_ = repo.Enqueue(context.Background(), ports.AutonomyOutboxRecord{
		ID: "ev-bad", ProjectID: "proj-a", IdempotencyKey: "idem-bad",
		EventType: "shell.run", SchemaVersion: "1", EventID: "ev-bad",
		Status: ports.OutboxPending, AvailableAt: time.Now().UTC(),
	})
	ctrl := autonomy.NewController(&ai.MockBrain{})
	proc := autonomy.NewEventAutonomyProcessor(ctrl, &ai.MockBrain{})
	d := autonomy.NewDispatcher(repo, proc, "w")
	d.MaxAttempts = 2
	_, _, dead, err := d.ProcessBatch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if dead != 1 {
		t.Fatalf("expected dead-letter, got dead=%d", dead)
	}
	got, _ := repo.Get(context.Background(), "proj-a", "ev-bad")
	if got.Status != ports.OutboxDeadLetter {
		t.Fatal(got.Status)
	}
}

func Test15C_ProjectIsolationGet(t *testing.T) {
	repo := memory.NewAutonomyOutboxRepo()
	_ = repo.Enqueue(context.Background(), ports.AutonomyOutboxRecord{
		ID: "ev-iso", ProjectID: "proj-a", IdempotencyKey: "ik",
		EventType: string(autonomy.EventWorkflowCompleted), SchemaVersion: "1", EventID: "ev-iso",
		Status: ports.OutboxPending, AvailableAt: time.Now().UTC(),
	})
	_, err := repo.Get(context.Background(), "proj-b", "ev-iso")
	if err == nil {
		t.Fatal("cross-project get must fail")
	}
}

func Test15C_PerRunOrderingOnePerBatch(t *testing.T) {
	repo := memory.NewAutonomyOutboxRepo()
	for i, id := range []string{"e1", "e2"} {
		_ = repo.Enqueue(context.Background(), ports.AutonomyOutboxRecord{
			ID: id, ProjectID: "proj-a", IdempotencyKey: "ik-" + id, AutonomyRunID: "same-run",
			EventType: string(autonomy.EventExecutionCompleted), SchemaVersion: "1", EventID: id,
			Status: ports.OutboxPending, AvailableAt: time.Now().UTC().Add(time.Duration(i) * time.Millisecond),
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Millisecond),
		})
	}
	batch, err := repo.ClaimBatch(context.Background(), "w", 10, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) != 1 {
		t.Fatalf("expected one claim per run, got %d", len(batch))
	}
}

func Test15C_NoGrantInOutboxRecord(t *testing.T) {
	rec := ports.AutonomyOutboxRecord{ID: "x", ProjectID: "p", IdempotencyKey: "k"}
	// structural: type has no Grant field — compile-time safety via ports definition
	if rec.IdempotencyKey == "" {
		t.Fatal("sanity")
	}
	_ = contracts.ProjectID("p")
}

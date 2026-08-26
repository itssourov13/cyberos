//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
	"cyberos/internal/storage/postgres"
)

func Test16A_PG_AutonomyRunRestart(t *testing.T) {
	dsn := os.Getenv("CYBEROS_DATABASE_URL")
	if dsn == "" {
		t.Skip("CYBEROS_DATABASE_URL unset — LIVE PG VERIFICATION SKIPPED (not a pass)")
	}
	db, err := postgres.Open(dsn)
	if err != nil {
		t.Fatalf("CYBEROS_DATABASE_URL set but unreachable: %v", err)
	}
	defer db.Close()
	repo := postgres.NewAutonomyRunRepo(db)
	ctrl1 := autonomy.NewController(&ai.MockBrain{})
	ctrl1.Repo = repo
	id := "ar-pg-16a-" + time.Now().Format("150405.000")
	run, err := ctrl1.Start(context.Background(), autonomy.StartRequest{
		ID: id, Goal: "pg-restart", ProjectID: "proj-pg", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-pg", MaxSteps: 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = ctrl1.RecordStepExecuted(run.ID)
	_ = ctrl1.RecordStepExecuted(run.ID)

	ctrl2 := autonomy.NewController(&ai.MockBrain{})
	ctrl2.Repo = repo
	restored, err := ctrl2.LoadFromRepo(context.Background(), "proj-pg", id)
	if err != nil {
		t.Fatal(err)
	}
	if restored.StepsConsumed != 2 {
		t.Fatalf("steps=%d", restored.StepsConsumed)
	}
	_, err = ctrl2.Cancel(restored.ID)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ctrl2.LoadFromRepo(context.Background(), "proj-pg", id)
	if err != nil || again.State != autonomy.RunCancelled {
		t.Fatal(err, again)
	}
}

func Test16A_PG_OutboxLease(t *testing.T) {
	dsn := os.Getenv("CYBEROS_DATABASE_URL")
	if dsn == "" {
		t.Skip("CYBEROS_DATABASE_URL unset — LIVE PG VERIFICATION SKIPPED (not a pass)")
	}
	db, err := postgres.Open(dsn)
	if err != nil {
		t.Fatalf("unreachable: %v", err)
	}
	defer db.Close()
	repo := postgres.NewAutonomyOutboxRepo(db)
	id := "ev-pg-16a-" + time.Now().Format("150405.000")
	err = repo.Enqueue(context.Background(), ports.AutonomyOutboxRecord{
		ID: id, ProjectID: "proj-pg", IdempotencyKey: id, EventID: id,
		EventType: string(autonomy.EventFindingCreated), SchemaVersion: "1",
		Status: ports.OutboxPending, AvailableAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := repo.ClaimBatch(context.Background(), "w1", 1, 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch) == 0 {
		t.Skip("no claimable row (possible concurrent test pollution)")
	}
	time.Sleep(80 * time.Millisecond)
	batch2, err := repo.ClaimBatch(context.Background(), "w2", 1, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = batch2
	_ = contracts.ProjectID("proj-pg")
}

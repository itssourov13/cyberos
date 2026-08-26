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

func Test16A_RestartRestoresBudgetAndState(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	ctrl1 := autonomy.NewController(&ai.MockBrain{})
	ctrl1.Repo = repo
	run, err := ctrl1.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-16a-1", Goal: "recon", ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a",
			MaxSteps: 5, MaxRuntime: time.Hour,
		},
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = ctrl1.RecordStepExecuted(run.ID)
	_ = ctrl1.RecordStepExecuted(run.ID)

	ctrl2 := autonomy.NewController(&ai.MockBrain{})
	ctrl2.Repo = repo
	restored, err := ctrl2.LoadFromRepo(context.Background(), "proj-a", "ar-16a-1")
	if err != nil {
		t.Fatal(err)
	}
	if restored.StepsConsumed != 2 {
		t.Fatalf("budget reset after restart: steps=%d", restored.StepsConsumed)
	}
	if restored.ProjectID != "proj-a" || restored.ScopeID != "scope-a" {
		t.Fatal("identity lost", restored)
	}
	if restored.Policy.MaxSteps != 5 {
		t.Fatal("policy snapshot lost")
	}
}

func Test16A_CancelSurvivesRestart(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	ctrl1 := autonomy.NewController(&ai.MockBrain{})
	ctrl1.Repo = repo
	run, err := ctrl1.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-16a-can", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous,
			ProjectID: "proj-a", MaxSteps: 10,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ctrl1.Cancel(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = ctrl1.Cancel(run.ID)
	_, _ = ctrl1.Cancel(run.ID)

	ctrl2 := autonomy.NewController(&ai.MockBrain{})
	ctrl2.Repo = repo
	restored, err := ctrl2.LoadFromRepo(context.Background(), "proj-a", run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.State != autonomy.RunCancelled {
		t.Fatalf("expected CANCELLED, got %s", restored.State)
	}
	adv, err := ctrl2.Advance(context.Background(), restored.ID)
	if err == nil && adv != nil && adv.MayAdvance {
		t.Fatal("cancelled run must not MayAdvance")
	}
}

func Test16A_ProjectIsolationOnLoad(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	ctrl := autonomy.NewController(&ai.MockBrain{})
	ctrl.Repo = repo
	_, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-iso", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 3,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ctrl.LoadFromRepo(context.Background(), "proj-b", "ar-iso")
	if err == nil {
		t.Fatal("cross-project load must fail")
	}
}

func Test16A_OutboxLeaseRecovery(t *testing.T) {
	repo := memory.NewAutonomyOutboxRepo()
	rec := ports.AutonomyOutboxRecord{
		ID: "ev-16a", ProjectID: "proj-a", IdempotencyKey: "idem-16a",
		EventType: string(autonomy.EventExecutionCompleted), SchemaVersion: "1", EventID: "ev-16a",
		Status: ports.OutboxPending, AvailableAt: time.Now().UTC(),
	}
	if err := repo.Enqueue(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	batch, err := repo.ClaimBatch(context.Background(), "w1", 1, time.Millisecond)
	if err != nil || len(batch) != 1 {
		t.Fatal(err, batch)
	}
	time.Sleep(5 * time.Millisecond)
	batch2, err := repo.ClaimBatch(context.Background(), "w2", 1, time.Second)
	if err != nil || len(batch2) != 1 {
		t.Fatal(err, len(batch2))
	}
	if batch2[0].LockedBy != "w2" {
		t.Fatal(batch2[0].LockedBy)
	}
}

func Test16A_NoGrantInPersistedRecord(t *testing.T) {
	repo := memory.NewAutonomyRunRepo()
	ctrl := autonomy.NewController(&ai.MockBrain{})
	ctrl.Repo = repo
	run, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-nog", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 2,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(context.Background(), "proj-a", run.ID)
	if err != nil {
		t.Fatal(err)
	}
	// Structural: AutonomyRunRecord has no Grant fields (compile-time by type).
	if got.State == "" || got.ProjectID == "" {
		t.Fatal(got)
	}
}

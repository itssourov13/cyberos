package assessment

import (
	"context"
	"errors"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/memory"
)

func TestAssessmentRepositoryClaimIsAtomic(t *testing.T) {
	repo := memory.NewAssessmentStore()
	a, err := New("claim-asmt-1", "project-a", "scope-a", "principal-a", "Atomic claim", "asset.example")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Plan([]contracts.AssessmentStep{{ID: "s1", CapabilityID: "c1"}}); err != nil {
		t.Fatal(err)
	}
	steps, _ := assessmentToRecord(a)
	if err := repo.Create(context.Background(), steps); err != nil {
		t.Fatal(err)
	}

	claimed, err := repo.ClaimRun(context.Background(), string(a.ID))
	if err != nil {
		t.Fatal(err)
	}
	if claimed.State != string(contracts.AssessmentStateRunning) {
		t.Fatalf("claim state=%q", claimed.State)
	}
	if _, err := repo.ClaimRun(context.Background(), string(a.ID)); !errors.Is(err, contracts.ErrConflict) {
		t.Fatalf("second replica must not claim an already-running assessment, got %v", err)
	}
}

func TestProductionAssessmentGetAlwaysReadsDurableState(t *testing.T) {
	repo := memory.NewAssessmentStore()
	orch := NewOrchestrator(staticIssuer{}, nil).WithRepository(repo)
	a, err := New("fresh-read-1", "project-a", "scope-a", "principal-a", "Fresh read", "asset.example")
	if err != nil {
		t.Fatal(err)
	}
	if err := orch.Save(a); err != nil {
		t.Fatal(err)
	}
	rec, err := repo.GetByID(context.Background(), string(a.ID))
	if err != nil {
		t.Fatal(err)
	}
	rec.Title = "Changed by another replica"
	if err := repo.Update(context.Background(), rec); err != nil {
		t.Fatal(err)
	}
	got, err := orch.Get(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != rec.Title {
		t.Fatalf("stale cached assessment returned: got %q want %q", got.Title, rec.Title)
	}
}

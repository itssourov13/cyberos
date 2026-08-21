//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"cyberos/internal/contracts"
	domainfinding "cyberos/internal/domain/finding"
	"cyberos/internal/findings"
	"cyberos/internal/riskprio"
	"cyberos/internal/storage/postgres"
)

func TestPhase7A_FindingsAndRiskDurability(t *testing.T) {
	db := setupDB(t) // migrates full chain including 007/009
	ctx := context.Background()

	fRepo := postgres.NewFindingRepo(db)
	rRepo := postgres.NewRiskEvalRepo(db)

	f, err := domainfinding.New("f-7a-1", "task-1", "proj-7a", "scope-1", contracts.FindingSeverityHigh, "Open port")
	if err != nil {
		t.Fatal(err)
	}
	f.Fingerprint = "fp-7a-unique"
	f.AssessmentID = "asmt-7a"
	f.Description = "22/tcp"
	if err := fRepo.Upsert(ctx, f); err != nil {
		t.Fatal(err)
	}

	eng := riskprio.NewEngine().WithRepository(rRepo)
	ev, err := eng.Evaluate(riskprio.Input{
		FindingID: f.ID, ProjectID: f.ProjectID, AssessmentID: f.AssessmentID,
		Severity: f.Severity, Confidence: contracts.FindingConfidenceHigh,
		AssetCriticality: contracts.AssetCriticalityHigh, Exposure: contracts.ExposureInternetFacing,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Restart: new connection + empty service maps
	_ = db.Close()
	db2 := setupDB(t)
	fRepo2 := postgres.NewFindingRepo(db2)
	rRepo2 := postgres.NewRiskEvalRepo(db2)

	got, err := fRepo2.GetByID(ctx, f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != f.Title || got.Fingerprint != f.Fingerprint {
		t.Fatalf("%+v", got)
	}
	byFP, err := fRepo2.GetByFingerprint(ctx, f.ProjectID, f.Fingerprint)
	if err != nil || byFP.ID != f.ID {
		t.Fatal(err, byFP)
	}

	rec, err := rRepo2.GetByFinding(ctx, f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Score != ev.Score || rec.Priority != ev.Priority {
		t.Fatalf("%+v vs %+v", rec, ev)
	}

	svc2 := findings.NewService(nil).WithRepository(fRepo2)
	loaded, err := svc2.Get(f.ID)
	if err != nil || loaded.Title != f.Title {
		t.Fatal(err, loaded)
	}
	eng2 := riskprio.NewEngine().WithRepository(rRepo2)
	loadedRisk, err := eng2.GetByFinding(f.ID)
	if err != nil || loadedRisk.Score != ev.Score {
		t.Fatal(err, loadedRisk)
	}
}

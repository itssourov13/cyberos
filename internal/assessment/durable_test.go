package assessment

import (
	"context"
	"errors"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution/mock"
	"cyberos/internal/storage/memory"
)

func TestAssessmentRepositoryReloadsStateAcrossOrchestrators(t *testing.T) {
	repo := memory.NewAssessmentStore()
	first := NewOrchestrator(staticIssuer{}, broker.New(mock.New())).WithRepository(repo)

	a, err := New("durable-asmt-1", "project-a", "scope-a", "principal-a", "Durable test", "asset-a.example")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Plan([]contracts.AssessmentStep{{
		ID: "step-1", CapabilityID: "assessment.stub.probe",
		ExecutorHint: "mock",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := first.Save(a); err != nil {
		t.Fatal(err)
	}

	second := NewOrchestrator(staticIssuer{}, broker.New(mock.New())).WithRepository(repo)
	loaded, err := second.Get(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State != contracts.AssessmentStatePlanned || len(loaded.Steps) != 1 || loaded.Target != a.Target {
		t.Fatalf("persisted assessment did not round-trip: %+v", loaded)
	}

	loaded.StepResults = []StepResult{{
		StepID: "step-1", CapabilityID: "assessment.stub.probe", Success: true,
		Summary: "completed before restart", GrantID: "grant-step-1", CompletedAt: time.Now().UTC(),
	}}
	if err := second.Save(loaded); err != nil {
		t.Fatal(err)
	}
	third := NewOrchestrator(staticIssuer{}, broker.New(mock.New())).WithRepository(repo)
	reloaded, err := third.Get(a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.StepResults) != 1 || !reloaded.StepResults[0].Success {
		t.Fatalf("completed step result was not persisted: %+v", reloaded.StepResults)
	}
}

func TestRunningAssessmentCannotBeAutomaticallyReplayedAfterRestart(t *testing.T) {
	repo := memory.NewAssessmentStore()
	first := NewOrchestrator(staticIssuer{}, broker.New(mock.New())).WithRepository(repo)
	a, err := New("durable-asmt-running", "project-a", "scope-a", "principal-a", "Interrupted run", "asset-a.example")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Plan([]contracts.AssessmentStep{{
		ID: "step-1", CapabilityID: "assessment.stub.probe", ExecutorHint: "mock",
	}}); err != nil {
		t.Fatal(err)
	}
	if err := a.TransitionTo(contracts.AssessmentStateRunning); err != nil {
		t.Fatal(err)
	}
	if err := first.Save(a); err != nil {
		t.Fatal(err)
	}

	afterRestart := NewOrchestrator(staticIssuer{}, broker.New(mock.New())).WithRepository(repo)
	_, err = afterRestart.Run(context.Background(), a.ID)
	if !errors.Is(err, contracts.ErrConflict) {
		t.Fatalf("an interrupted RUNNING assessment must not auto replay, got %v", err)
	}
}

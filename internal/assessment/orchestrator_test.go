package assessment

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/events"
	"cyberos/internal/evidence"
	"cyberos/internal/execution/mock"
	"cyberos/internal/execution/sandbox"
	"cyberos/internal/findings"
	"cyberos/internal/provenance"
)

type staticIssuer struct {
	deny bool
}

func (s staticIssuer) IssueForStep(_ context.Context, a *Assessment, step contracts.AssessmentStep) (contracts.ExecutionGrant, error) {
	if s.deny {
		return contracts.ExecutionGrant{}, fmt.Errorf("%w: denied", contracts.ErrForbidden)
	}
	now := time.Now().UTC()
	return contracts.ExecutionGrant{
		ID:        contracts.ExecutionGrantID("grant-" + step.ID),
		Principal: contracts.PrincipalRef{ID: a.PrincipalID, Type: contracts.PrincipalTypeHuman},
		ProjectID: a.ProjectID, ScopeID: a.ScopeID,
		TaskID: contracts.TaskID("task-" + string(a.ID)), CapabilityID: step.CapabilityID,
		Status:   contracts.ExecutionGrantStatusIssued,
		IssuedAt: now.Add(-time.Minute), ExpiresAt: now.Add(5 * time.Minute),
		Nonce: "nonce-" + step.ID, CorrelationID: contracts.CorrelationID("corr-" + step.ID),
	}, nil
}

func TestAssessmentPlanAndRun(t *testing.T) {
	brk := broker.New(sandbox.New(mock.New()))
	ev := evidence.NewService()
	orch := NewOrchestrator(staticIssuer{}, brk).WithEvidence(ev, findings.NewService(ev), provenance.NewService(), events.NewStore())

	a, err := New("asmt-1", "proj-1", "scope-1", "user-1", "Demo", "example.test")
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Plan([]contracts.AssessmentStep{
		{ID: "s1", CapabilityID: "assessment.stub.probe", Parameters: map[string]any{"x": 1}},
		{ID: "s2", CapabilityID: "assessment.stub.probe"},
	}); err != nil {
		t.Fatal(err)
	}
	_ = orch.Save(a)
	out, err := orch.Run(context.Background(), "asmt-1")
	if err != nil {
		t.Fatal(err)
	}
	if out.State != contracts.AssessmentStateSucceeded {
		t.Fatalf("state=%s results=%+v", out.State, out.StepResults)
	}
	if len(out.StepResults) != 2 || !out.StepResults[0].Success {
		t.Fatalf("%+v", out.StepResults)
	}
}

func TestAssessmentGrantDeniedFailsClosed(t *testing.T) {
	orch := NewOrchestrator(staticIssuer{deny: true}, broker.New(mock.New()))
	a, _ := New("asmt-2", "p", "s", "u", "T", "t")
	_ = a.Plan([]contracts.AssessmentStep{{ID: "s1", CapabilityID: "c"}})
	_ = orch.Save(a)
	out, err := orch.Run(context.Background(), "asmt-2")
	if err != nil {
		t.Fatal(err)
	}
	if out.State != contracts.AssessmentStateFailed {
		t.Fatalf("state=%s", out.State)
	}
	if out.StepResults[0].Success {
		t.Fatal("step should fail without grant")
	}
}

func TestCancel(t *testing.T) {
	a, _ := New("asmt-3", "p", "s", "u", "T", "t")
	_ = a.Plan([]contracts.AssessmentStep{{ID: "s1", CapabilityID: "c"}})
	if err := a.Cancel(); err != nil {
		t.Fatal(err)
	}
	if a.State != contracts.AssessmentStateCancelled {
		t.Fatal(a.State)
	}
}

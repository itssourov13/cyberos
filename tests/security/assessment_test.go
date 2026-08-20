package security_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cyberos/internal/assessment"
	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution/mock"
)

type denyIssuer struct{}

func (denyIssuer) IssueForStep(context.Context, *assessment.Assessment, contracts.AssessmentStep) (contracts.ExecutionGrant, error) {
	return contracts.ExecutionGrant{}, fmt.Errorf("no grant")
}

// Orchestration cannot execute without per-step grants.
func TestSecurity_AssessmentRequiresPerStepGrant(t *testing.T) {
	orch := assessment.NewOrchestrator(denyIssuer{}, broker.New(mock.New()))
	a, _ := assessment.New("a1", "p", "s", "u", "title", "target")
	_ = a.Plan([]contracts.AssessmentStep{{ID: "1", CapabilityID: "cap"}})
	_ = orch.Save(a)
	out, err := orch.Run(context.Background(), "a1")
	if err != nil {
		t.Fatal(err)
	}
	if out.State != contracts.AssessmentStateFailed {
		t.Fatalf("state=%s", out.State)
	}
}

func TestSecurity_AssessmentBoundToScope(t *testing.T) {
	a, err := assessment.New("a2", "proj", "scope-x", "user", "t", "host")
	if err != nil {
		t.Fatal(err)
	}
	if a.ScopeID != "scope-x" || a.ProjectID != "proj" {
		t.Fatal("must bind scope/project")
	}
	_ = time.Now()
}

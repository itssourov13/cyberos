package security_test

import (
	"context"
	"testing"

	"cyberos/internal/assessment"
	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/execution/mock"
	"cyberos/internal/sdk"
)

func TestSecurity_Phase3J_NoGrantNoAssessmentStep(t *testing.T) {
	orch := assessment.NewOrchestrator(denyIssuer{}, broker.New(mock.New()))
	a, _ := assessment.New("x", "p", "s", "u", "t", "tgt")
	_ = a.Plan([]contracts.AssessmentStep{{ID: "1", CapabilityID: "c"}})
	_ = orch.Save(a)
	out, _ := orch.Run(context.Background(), "x")
	if out.State != contracts.AssessmentStateFailed {
		t.Fatalf("state=%s", out.State)
	}
}

func TestSecurity_Phase3J_SDKRequiresWireForRun(t *testing.T) {
	c := sdk.NewInMemory()
	// Without WireAssessment + CP, Create may fail
	_, err := c.CreateAssessment("a", "p", "s", "u", "t", "")
	if err == nil {
		// Create works only if Assessment set — NewInMemory does not WireAssessment by default
		t.Fatal("expected error without orchestrator")
	}
}

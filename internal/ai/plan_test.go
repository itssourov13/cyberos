package ai_test

import (
	"context"
	"testing"

	"cyberos/internal/ai"
	"cyberos/internal/contracts"
)

func validPlan() *ai.Plan {
	return &ai.Plan{
		ID: "p1", ContractVersion: ai.PlanContractV1, Goal: "scan",
		ProjectID: "proj", ScopeID: "scope", TenantID: "ten", PrincipalID: "user",
		Steps: []ai.PlanStep{{ID: "s1", CapabilityID: "recon.network.portscan", Parameters: map[string]any{"target": "example.com"}}},
	}
}

func TestValidPlan(t *testing.T) {
	if err := ai.ValidatePlan(validPlan()); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownVersion(t *testing.T) {
	p := validPlan()
	p.ContractVersion = "99"
	if err := ai.ValidatePlan(p); err == nil {
		t.Fatal("version")
	}
}

func TestDuplicateStepID(t *testing.T) {
	p := validPlan()
	p.Steps = append(p.Steps, ai.PlanStep{ID: "s1", CapabilityID: "web.http.probe"})
	if err := ai.ValidatePlan(p); err == nil {
		t.Fatal("dup")
	}
}

func TestMissingDependency(t *testing.T) {
	p := validPlan()
	p.Steps[0].DependsOn = []string{"missing"}
	if err := ai.ValidatePlan(p); err == nil {
		t.Fatal("dep")
	}
}

func TestDependencyCycle(t *testing.T) {
	p := validPlan()
	p.Steps = []ai.PlanStep{
		{ID: "a", CapabilityID: "recon.network.portscan", DependsOn: []string{"b"}},
		{ID: "b", CapabilityID: "web.http.probe", DependsOn: []string{"a"}},
	}
	if err := ai.ValidatePlan(p); err == nil {
		t.Fatal("cycle")
	}
}

func TestEmptyCapability(t *testing.T) {
	p := validPlan()
	p.Steps[0].CapabilityID = ""
	if err := ai.ValidatePlan(p); err == nil {
		t.Fatal("cap")
	}
}

func TestRejectExecutableCapability(t *testing.T) {
	p := validPlan()
	p.Steps[0].CapabilityID = "nmap"
	if err := ai.ValidatePlan(p); err == nil {
		t.Fatal("nmap as capability")
	}
	p.Steps[0].CapabilityID = "/usr/bin/nmap"
	if err := ai.ValidatePlan(p); err == nil {
		t.Fatal("path")
	}
}

func TestRejectForbiddenParams(t *testing.T) {
	p := validPlan()
	p.Steps[0].Parameters = map[string]any{"binary": "/bin/sh"}
	if err := ai.ValidatePlan(p); err == nil {
		t.Fatal("binary")
	}
	p.Steps[0].Parameters = map[string]any{"target": "a && b"}
	if err := ai.ValidatePlan(p); err == nil {
		t.Fatal("shell")
	}
}

func TestProjectScopePreserved(t *testing.T) {
	p := validPlan()
	if p.ProjectID != "proj" || p.ScopeID != "scope" || p.TenantID != "ten" {
		t.Fatal(p)
	}
}

func TestMockBrainNoNetwork(t *testing.T) {
	b := &ai.MockBrain{}
	resp, err := b.Plan(context.Background(), ai.BrainRequest{
		Operation: ai.OpPlan, Objective: "map network", ProjectID: "p", ScopeID: "s",
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil || resp.Plan == nil {
		t.Fatal(err, resp)
	}
	if err := ai.ValidatePlan(resp.Plan); err != nil {
		t.Fatal(err)
	}
	if resp.Plan.Goal != "map network" {
		t.Fatal(resp.Plan.Goal)
	}
}

func TestToWorkflowShapeStructuralOnly(t *testing.T) {
	shape, err := ai.ToWorkflowShape(validPlan())
	if err != nil || len(shape) != 1 {
		t.Fatal(err, shape)
	}
	if shape[0].CapabilityID != "recon.network.portscan" {
		t.Fatal(shape)
	}
}

func TestPlanValidationDoesNotAuthorize(t *testing.T) {
	if err := ai.ValidatePlan(validPlan()); err != nil {
		t.Fatal(err)
	}
}

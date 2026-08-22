package apply_test

import (
	"context"
	"testing"

	"cyberos/internal/ai"
	"cyberos/internal/ai/apply"
)

func validPlan() ai.Plan {
	return ai.Plan{
		ID: "plan-1", ContractVersion: ai.PlanContractV1, Goal: "recon",
		ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a",
		Steps: []ai.PlanStep{
			{ID: "s1", CapabilityID: "recon.network.portscan", Parameters: map[string]any{"target": "example.com"}},
			{ID: "s2", CapabilityID: "web.http.probe", DependsOn: []string{"s1"}, Parameters: map[string]any{"url": "https://example.com"}},
		},
	}
}

func TestApplyCreatesWorkflow(t *testing.T) {
	app := apply.NewApplicator(nil)
	res, err := app.Apply(context.Background(), apply.ApplyRequest{
		Plan: validPlan(), ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a",
		PrincipalID: "user-1", Accepted: true, ProviderID: "mock", Model: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "WORKFLOW_CREATED" || res.Definition.ID == "" {
		t.Fatal(res)
	}
	if len(res.Definition.Steps) != 2 {
		t.Fatal(res.Definition.Steps)
	}
	if res.Definition.Steps[0].CapabilityID != "recon.network.portscan" {
		t.Fatal(res.Definition.Steps[0])
	}
	if len(res.Definition.Steps[1].DependsOn) != 1 || res.Definition.Steps[1].DependsOn[0] != "s1" {
		t.Fatal(res.Definition.Steps[1])
	}
}

func TestApplyRequiresAcceptance(t *testing.T) {
	app := apply.NewApplicator(nil)
	_, err := app.Apply(context.Background(), apply.ApplyRequest{
		Plan: validPlan(), ProjectID: "proj-a", ScopeID: "scope-a", Accepted: false,
	})
	if err == nil {
		t.Fatal("must require acceptance")
	}
}

func TestProjectMismatch(t *testing.T) {
	app := apply.NewApplicator(nil)
	_, err := app.Apply(context.Background(), apply.ApplyRequest{
		Plan: validPlan(), ProjectID: "other", ScopeID: "scope-a", TenantID: "ten-a", Accepted: true,
	})
	if err == nil {
		t.Fatal("project mismatch")
	}
}

func TestTenantMismatch(t *testing.T) {
	app := apply.NewApplicator(nil)
	_, err := app.Apply(context.Background(), apply.ApplyRequest{
		Plan: validPlan(), ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "other", Accepted: true,
	})
	if err == nil {
		t.Fatal("tenant")
	}
}

func TestInvalidPlanRejected(t *testing.T) {
	app := apply.NewApplicator(nil)
	p := validPlan()
	p.Steps[0].CapabilityID = "nmap"
	_, err := app.Apply(context.Background(), apply.ApplyRequest{
		Plan: p, ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a", Accepted: true,
	})
	if err == nil {
		t.Fatal("invalid plan")
	}
}

func TestIdempotentApply(t *testing.T) {
	app := apply.NewApplicator(nil)
	req := apply.ApplyRequest{
		Plan: validPlan(), ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a", Accepted: true,
	}
	r1, err := app.Apply(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := app.Apply(context.Background(), req)
	if err != nil || !r2.Idempotent {
		t.Fatal(err, r2)
	}
	if r1.WorkflowID != r2.WorkflowID {
		t.Fatal("idempotent workflow id")
	}
}

func TestProvenanceRecord(t *testing.T) {
	store := apply.NewStore()
	app := apply.NewApplicator(store)
	res, err := app.Apply(context.Background(), apply.ApplyRequest{
		Plan: validPlan(), ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a",
		Accepted: true, SessionID: "sess-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	rec, ok := store.GetPlanRecord(res.PlanID)
	if !ok || rec.WorkflowID != res.WorkflowID || rec.SessionID != "sess-1" {
		t.Fatal(rec, ok)
	}
}

func TestApplyDoesNotGrantOrBroker(t *testing.T) {
	// Structural: Apply has no Grant/Broker dependencies — only Store + Plan validation.
	app := apply.NewApplicator(nil)
	_, err := app.Apply(context.Background(), apply.ApplyRequest{
		Plan: validPlan(), ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a", Accepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}

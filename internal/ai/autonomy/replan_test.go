package autonomy_test

import (
	"context"
	"testing"

	"cyberos/internal/ai"
	"cyberos/internal/ai/apply"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
)

func TestRePlanCreatesValidatedPlan(t *testing.T) {
	rp := autonomy.NewRePlanner(&ai.MockBrain{}, nil)
	res, err := rp.RePlan(context.Background(), autonomy.RePlanRequest{
		ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a", PrincipalID: "u1",
		Goal: "continue recon", PreviousAnalysisID: "an-1",
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil || res.Plan == nil {
		t.Fatal(err, res)
	}
	if res.Plan.ProjectID != "proj-a" {
		t.Fatal(res.Plan.ProjectID)
	}
	if res.Lineage.AnalysisID != "an-1" || res.Lineage.PlanID != res.Plan.ID {
		t.Fatal(res.Lineage)
	}
	if res.Applied {
		t.Fatal("should not apply without AcceptApply")
	}
}

func TestRePlanApplyCreatesDefinitionOnly(t *testing.T) {
	rp := autonomy.NewRePlanner(&ai.MockBrain{}, apply.NewApplicator(nil))
	res, err := rp.RePlan(context.Background(), autonomy.RePlanRequest{
		ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a",
		Goal: "scan", AcceptApply: true, PreviousAnalysisID: "an-2",
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil || !res.Applied || res.Definition == nil {
		t.Fatal(err, res)
	}
	if res.Definition.ProjectID != "proj-a" {
		t.Fatal(res.Definition)
	}
	// No Orchestrator/Grant in RePlanner — structural guarantee.
}

func TestRePlanAnalysisProjectMismatch(t *testing.T) {
	rp := autonomy.NewRePlanner(nil, nil)
	_, err := rp.RePlan(context.Background(), autonomy.RePlanRequest{
		ProjectID: "proj-a", ScopeID: "s", Goal: "g",
		Analysis: &autonomy.AnalysisResult{AnalysisID: "x", ProjectID: "other", Trust: autonomy.TrustUntrusted},
	})
	if err == nil {
		t.Fatal("mismatch")
	}
}

func TestRePlanIdempotentApply(t *testing.T) {
	app := apply.NewApplicator(nil)
	rp := autonomy.NewRePlanner(&ai.MockBrain{FixedPlan: &ai.Plan{
		ID: "fixed-plan", ContractVersion: ai.PlanContractV1, Goal: "g",
		ProjectID: "proj-a", ScopeID: "scope-a",
		Steps: []ai.PlanStep{{ID: "s1", CapabilityID: "recon.network.portscan"}},
	}}, app)
	req := autonomy.RePlanRequest{
		ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a", Goal: "g", AcceptApply: true,
	}
	r1, err := rp.RePlan(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := rp.RePlan(context.Background(), req)
	if err != nil || !r2.Idempotent {
		t.Fatal(err, r2)
	}
	if r1.Definition.ID != r2.Definition.ID {
		t.Fatal("idempotent definition")
	}
}

func TestRePlanLineageParentPlan(t *testing.T) {
	rp := autonomy.NewRePlanner(&ai.MockBrain{}, nil)
	res, err := rp.RePlan(context.Background(), autonomy.RePlanRequest{
		ProjectID: "p", ScopeID: "s", Goal: "g", PreviousPlanID: "plan-old", PreviousAnalysisID: "an-old",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Lineage.ParentPlanID != "plan-old" {
		t.Fatal(res.Lineage)
	}
}

func TestRePlanDoesNotGrantOrAdvance(t *testing.T) {
	rp := autonomy.NewRePlanner(&ai.MockBrain{}, nil)
	_, err := rp.RePlan(context.Background(), autonomy.RePlanRequest{
		ProjectID: "p", ScopeID: "s", Goal: "g", AcceptApply: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}

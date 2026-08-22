package autonomy_test

import (
	"testing"
	"time"

	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
)

func basePolicy(mode autonomy.AutonomyMode) autonomy.AutonomyPolicy {
	return autonomy.AutonomyPolicy{
		Version: autonomy.PolicyContractV1, Mode: mode, ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		AllowedCapabilities: []contracts.CapabilityID{"recon.network.portscan"},
		MaxSteps:            5, MaxRuntime: time.Hour, MaxConcurrentSteps: 2,
	}
}

func TestManualWaits(t *testing.T) {
	d, err := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		CapabilityID: "recon.network.portscan", Policy: basePolicy(autonomy.ModeManual),
	})
	if err != nil || d.Decision != autonomy.DecisionWait {
		t.Fatal(err, d)
	}
	if autonomy.MayAutoAdvance(d) {
		t.Fatal("manual must not auto advance")
	}
}

func TestAssistedWaits(t *testing.T) {
	d, _ := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		CapabilityID: "recon.network.portscan", Policy: basePolicy(autonomy.ModeAssisted),
	})
	if d.Decision != autonomy.DecisionWait {
		t.Fatal(d)
	}
}

func TestControlledAllows(t *testing.T) {
	d, err := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		CapabilityID: "recon.network.portscan", Policy: basePolicy(autonomy.ModeControlledAutonomous),
	})
	if err != nil || d.Decision != autonomy.DecisionAllow {
		t.Fatal(err, d)
	}
}

func TestUnknownModeFails(t *testing.T) {
	p := basePolicy(autonomy.ModeManual)
	p.Mode = "GOD"
	_, err := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", Policy: p,
	})
	if err == nil {
		t.Fatal("unknown mode")
	}
}

func TestProjectMismatch(t *testing.T) {
	d, _ := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "other", TenantID: "ten-a", ScopeID: "scope-a",
		CapabilityID: "recon.network.portscan", Policy: basePolicy(autonomy.ModeControlledAutonomous),
	})
	if d.Decision != autonomy.DecisionDeny || d.Reason != "project_mismatch" {
		t.Fatal(d)
	}
}

func TestDenyOverridesAllow(t *testing.T) {
	p := basePolicy(autonomy.ModeControlledAutonomous)
	p.DeniedCapabilities = []contracts.CapabilityID{"recon.network.portscan"}
	d, _ := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		CapabilityID: "recon.network.portscan", Policy: p,
	})
	if d.Decision != autonomy.DecisionDeny {
		t.Fatal(d)
	}
}

func TestMaxStepsStop(t *testing.T) {
	d, _ := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		CapabilityID: "recon.network.portscan", StepsExecuted: 5,
		Policy: basePolicy(autonomy.ModeControlledAutonomous),
	})
	if d.Decision != autonomy.DecisionStop {
		t.Fatal(d)
	}
}

func TestRiskThreshold(t *testing.T) {
	p := basePolicy(autonomy.ModeControlledAutonomous)
	p.MaxRisk = "medium"
	d, _ := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		CapabilityID: "recon.network.portscan", RiskLevel: "critical", Policy: p,
	})
	if d.Decision != autonomy.DecisionDeny {
		t.Fatal(d)
	}
}

func TestPolicyNotGrant(t *testing.T) {
	// Structural: EvaluateAutonomyPolicy has no Grant/Broker types.
	d, _ := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		CapabilityID: "recon.network.portscan", Policy: basePolicy(autonomy.ModeControlledAutonomous),
	})
	if d.Decision != autonomy.DecisionAllow {
		t.Fatal(d)
	}
}

func TestUnknownCapabilityDeniedWhenAllowlistSet(t *testing.T) {
	d, _ := autonomy.EvaluateAutonomyPolicy(autonomy.PolicyEvalRequest{
		ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		CapabilityID: "evil.cap", Policy: basePolicy(autonomy.ModeControlledAutonomous),
	})
	if d.Decision != autonomy.DecisionDeny {
		t.Fatal(d)
	}
}

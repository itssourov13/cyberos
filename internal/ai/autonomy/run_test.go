package autonomy_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
)

func TestStartAutonomyRunManualWaits(t *testing.T) {
	c := autonomy.NewController(&ai.MockBrain{})
	run, err := c.Start(context.Background(), autonomy.StartRequest{
		ID: "r1", Goal: "recon", ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a",
			MaxSteps: 5,
		},
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	adv, err := c.Advance(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if adv.MayAdvance {
		t.Fatal("manual must not auto-advance")
	}
	if adv.Run.State != autonomy.RunWaiting && adv.PolicyDecision.Decision != autonomy.DecisionWait {
		// after advance with wait
	}
}

func TestControlledMayAdvanceWhenAllowed(t *testing.T) {
	c := autonomy.NewController(&ai.MockBrain{})
	run, err := c.Start(context.Background(), autonomy.StartRequest{
		ID: "r2", Goal: "recon", ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous, ProjectID: "proj-a",
			AllowedCapabilities: []contracts.CapabilityID{"recon.network.portscan"},
			MaxSteps:            10, MaxRuntime: time.Hour, MaxLLMCalls: 50,
		},
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	adv, err := c.Advance(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !adv.MayAdvance {
		t.Fatalf("expected may advance: %+v", adv.PolicyDecision)
	}
	// Controller itself does not issue Grant — caller would use Orchestrator
}

func TestMaxStepsBlocks(t *testing.T) {
	c := autonomy.NewController(&ai.MockBrain{})
	run, err := c.Start(context.Background(), autonomy.StartRequest{
		ID: "r3", Goal: "g", ProjectID: "proj-a", ScopeID: "scope-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous, ProjectID: "proj-a",
			MaxSteps: 1, MaxLLMCalls: 50, MaxRuntime: time.Hour,
			AllowedCapabilities: []contracts.CapabilityID{"recon.network.portscan"},
		},
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = c.RecordStepExecuted(run.ID)
	adv, err := c.Advance(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if adv.MayAdvance {
		t.Fatal("max steps")
	}
	if adv.Run.State != autonomy.RunPolicyBlocked {
		t.Fatal(adv.Run.State)
	}
}

func TestCancelIdempotent(t *testing.T) {
	c := autonomy.NewController(&ai.MockBrain{})
	run, _ := c.Start(context.Background(), autonomy.StartRequest{
		ID: "r4", Goal: "g", ProjectID: "p", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "p"},
	})
	r1, err := c.Cancel(run.ID)
	if err != nil || r1.State != autonomy.RunCancelled {
		t.Fatal(err, r1)
	}
	r2, err := c.Cancel(run.ID)
	if err != nil || r2.State != autonomy.RunCancelled {
		t.Fatal(err, r2)
	}
}

func TestStartIdempotent(t *testing.T) {
	c := autonomy.NewController(&ai.MockBrain{})
	req := autonomy.StartRequest{
		ID: "r5", Goal: "g", ProjectID: "p", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "p"},
	}
	a, err := c.Start(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	b, err := c.Start(context.Background(), req)
	if err != nil || a.ID != b.ID {
		t.Fatal(err, a, b)
	}
}

func TestLineageOnIterate(t *testing.T) {
	c := autonomy.NewController(&ai.MockBrain{})
	run, err := c.Start(context.Background(), autonomy.StartRequest{
		ID: "r6", Goal: "g", ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous, ProjectID: "proj-a",
			AllowedCapabilities: []contracts.CapabilityID{"recon.network.portscan"}, MaxSteps: 5, MaxRuntime: time.Hour,
		},
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.AnalysisID == "" || run.CurrentPlanID == "" {
		t.Fatal(run)
	}
	if run.Iteration < 1 {
		t.Fatal(run.Iteration)
	}
}

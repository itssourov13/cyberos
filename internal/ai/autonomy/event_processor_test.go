package autonomy_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
)

func baseEvent() autonomy.AutonomyEvent {
	return autonomy.AutonomyEvent{
		EventID: "e15b-1", SchemaVersion: autonomy.AutonomyEventContractV1,
		Type: autonomy.EventExecutionCompleted, ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a",
		AutonomyRunID: "ar-15b", CorrelationID: "c1", IdempotencyKey: "idem-15b-1",
		Timestamp: time.Now().UTC(),
	}
}

func Test15B_ProcessManualWait(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-15b", Goal: "recon", ProjectID: "proj-a", ScopeID: "scope-a", TenantID: "ten-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 5,
		},
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = run
	p := autonomy.NewEventAutonomyProcessor(ctrl, &ai.MockBrain{})
	res, err := p.Process(context.Background(), baseEvent())
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != autonomy.ProcessPolicyWait && res.Status != autonomy.ProcessOK {
		// MANUAL → WAIT
		if res.AdvanceEligible {
			t.Fatal("MANUAL must not be eligible", res.Status)
		}
	}
	if res.AdvanceEligible {
		t.Fatal("MANUAL advance eligible")
	}
	if res.AnalysisID == "" || res.PlanID == "" {
		t.Fatal("expected analysis and plan", res)
	}
}

func Test15B_ControlledEligibleNoExecution(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	_, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-15b-c", Goal: "recon", ProjectID: "proj-a", ScopeID: "scope-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous,
			ProjectID: "proj-a", AllowedCapabilities: []contracts.CapabilityID{"recon.network.portscan"},
			MaxSteps: 10, MaxRuntime: time.Hour, MaxLLMCalls: 50,
		},
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ev := baseEvent()
	ev.AutonomyRunID = "ar-15b-c"
	ev.IdempotencyKey = "idem-ctrl-1"
	ev.EventID = "e-ctrl-1"
	p := autonomy.NewEventAutonomyProcessor(ctrl, &ai.MockBrain{})
	res, err := p.Process(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	// May be eligible; must not have called orchestrator (processor doesn't hold one).
	if res.AnalysisID == "" {
		t.Fatal(res)
	}
}

func Test15B_CancelledTerminal(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-can", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous, ProjectID: "proj-a", MaxSteps: 5, MaxLLMCalls: 50},
	})
	_, _ = ctrl.Cancel(run.ID)
	ev := baseEvent()
	ev.AutonomyRunID = run.ID
	ev.IdempotencyKey = "idem-can"
	ev.EventID = "e-can"
	p := autonomy.NewEventAutonomyProcessor(ctrl, &ai.MockBrain{})
	res, err := p.Process(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != autonomy.ProcessTerminal {
		t.Fatal(res.Status)
	}
	if res.AdvanceEligible {
		t.Fatal("cancelled")
	}
}

func Test15B_ProjectMismatchStale(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	_, _ = ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-stale", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 5},
	})
	ev := baseEvent()
	ev.AutonomyRunID = "ar-stale"
	ev.ProjectID = "proj-b" // mismatch vs run
	ev.IdempotencyKey = "idem-stale"
	ev.EventID = "e-stale"
	// validation requires project on event — run check happens after validate
	p := autonomy.NewEventAutonomyProcessor(ctrl, &ai.MockBrain{})
	res, err := p.Process(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != autonomy.ProcessStale {
		t.Fatal(res.Status, res.Error)
	}
}

func Test15B_IdempotentSkip(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	_, _ = ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-idem", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 5},
	})
	ev := baseEvent()
	ev.AutonomyRunID = "ar-idem"
	ev.IdempotencyKey = "same-key"
	ev.EventID = "e1"
	p := autonomy.NewEventAutonomyProcessor(ctrl, &ai.MockBrain{})
	r1, err := p.Process(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	ev.EventID = "e2" // different event id, same idempotency
	r2, err := p.Process(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	if r2.Status != autonomy.ProcessIdempotentSkip {
		t.Fatal(r1.Status, r2.Status)
	}
}

func Test15B_UnknownEventTypeRejected(t *testing.T) {
	p := autonomy.NewEventAutonomyProcessor(nil, &ai.MockBrain{})
	ev := baseEvent()
	ev.Type = "shell.run"
	_, err := p.Process(context.Background(), ev)
	if err == nil {
		t.Fatal("expected reject")
	}
}

func Test15B_ConcurrentProcessSafe(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	_, _ = ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "ar-conc", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 5},
	})
	p := autonomy.NewEventAutonomyProcessor(ctrl, &ai.MockBrain{})
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ev := baseEvent()
			ev.AutonomyRunID = "ar-conc"
			ev.EventID = "e-c-" + string(rune('a'+i))
			ev.IdempotencyKey = "conc-key-" + string(rune('a'+i))
			_, _ = p.Process(context.Background(), ev)
		}(i)
	}
	wg.Wait()
}

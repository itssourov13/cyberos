package autonomy_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
	"cyberos/internal/workflow"
)

type countingAuth struct {
	n atomic.Int32
}

func (c *countingAuth) AuthorizeStep(ctx context.Context, req workflow.AuthorizeStepRequest) (contracts.ExecutionGrant, error) {
	c.n.Add(1)
	return contracts.ExecutionGrant{
		ID:           contracts.ExecutionGrantID("g-" + string(req.TaskID)),
		Principal:    contracts.PrincipalRef{ID: req.PrincipalID},
		ProjectID:    req.ProjectID,
		ScopeID:      req.ScopeID,
		CapabilityID: req.CapabilityID,
		TaskID:       req.TaskID,
		Status:       contracts.ExecutionGrantStatusIssued,
		IssuedAt:     time.Now().UTC(),
		ExpiresAt:    time.Now().Add(time.Hour).UTC(),
	}, nil
}

func TestHandoffManualNoOrchestrator(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, err := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "h1", Goal: "g", ProjectID: "proj-a", ScopeID: "scope-a",
		Policy: autonomy.AutonomyPolicy{
			Version: autonomy.PolicyContractV1, Mode: autonomy.ModeManual, ProjectID: "proj-a", MaxSteps: 5,
		},
		CapabilityHints: []contracts.CapabilityID{"recon.network.portscan"},
	})
	if err != nil {
		t.Fatal(err)
	}
	auth := &countingAuth{}
	h := &autonomy.AdvanceHandoff{Controller: ctrl, Orchestrator: workflow.NewOrchestrator(nil, nil, nil, auth)}
	res, err := h.Execute(context.Background(), autonomy.HandoffRequest{RunID: run.ID, ProjectID: "proj-a"})
	if err != nil {
		t.Fatal(err)
	}
	if res.OrchestratorCalled || res.MayAdvance {
		t.Fatal(res)
	}
	if auth.n.Load() != 0 {
		t.Fatal("authorizer must not run on policy deny/wait")
	}
}

func TestHandoffCancelled(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "h3", Goal: "g", ProjectID: "p", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous, ProjectID: "p", MaxSteps: 5, MaxLLMCalls: 50},
	})
	_, _ = ctrl.Cancel(run.ID)
	h := &autonomy.AdvanceHandoff{Controller: ctrl}
	res, err := h.Execute(context.Background(), autonomy.HandoffRequest{RunID: run.ID, ProjectID: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if res.OrchestratorCalled {
		t.Fatal("cancelled")
	}
}

func TestHandoffProjectMismatch(t *testing.T) {
	ctrl := autonomy.NewController(&ai.MockBrain{})
	run, _ := ctrl.Start(context.Background(), autonomy.StartRequest{
		ID: "h4", Goal: "g", ProjectID: "proj-a", ScopeID: "s",
		Policy: autonomy.AutonomyPolicy{Version: autonomy.PolicyContractV1, Mode: autonomy.ModeControlledAutonomous, ProjectID: "proj-a", MaxSteps: 5, MaxLLMCalls: 50},
	})
	h := &autonomy.AdvanceHandoff{Controller: ctrl}
	res, err := h.Execute(context.Background(), autonomy.HandoffRequest{RunID: run.ID, ProjectID: "other"})
	if err != nil {
		t.Fatal(err)
	}
	if res.PolicyDecision.Reason != "project_mismatch" {
		t.Fatal(res.PolicyDecision)
	}
}

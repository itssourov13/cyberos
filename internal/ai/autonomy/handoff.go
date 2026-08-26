package autonomy

import (
	"context"
	"fmt"

	"cyberos/internal/contracts"
	"cyberos/internal/workflow"
)

// AdvanceHandoff delegates policy-allowed autonomous progression to the existing
// Workflow Orchestrator. It never issues Grants or calls Broker.
type AdvanceHandoff struct {
	Controller   *Controller
	Orchestrator *workflow.Orchestrator
}

// HandoffRequest is the authoritative context for a controlled advance attempt.
type HandoffRequest struct {
	RunID         string
	ProjectID     contracts.ProjectID
	TenantID      contracts.TenantID
	ScopeID       contracts.ScopeID
	WorkflowRunID string // optional; Orchestrator run id if already started
}

// HandoffResult reports policy decision and whether Orchestrator was invoked.
type HandoffResult struct {
	PolicyDecision     PolicyEvalResult
	OrchestratorCalled bool
	MayAdvance         bool
	Run                *AutonomyRun
	WorkflowRun        *workflow.Run
	Err                error
}

// Execute checks policy and, only when CONTROLLED_AUTONOMOUS + ALLOW, calls
// Orchestrator.Advance. CP→Grant remains inside the Orchestrator.
func (h *AdvanceHandoff) Execute(ctx context.Context, req HandoffRequest) (*HandoffResult, error) {
	if h == nil || h.Controller == nil {
		return nil, fmt.Errorf("%w: handoff not configured", contracts.ErrInvalidArgument)
	}
	out := &HandoffResult{}

	run, err := h.Controller.Get(req.RunID)
	if err != nil {
		// try durable load
		if h.Controller.Repo != nil && req.ProjectID != "" {
			run, err = h.Controller.LoadFromRepo(ctx, req.ProjectID, req.RunID)
		}
		if err != nil {
			return nil, err
		}
	}
	out.Run = run

	if run.State == RunCancelled || run.State == RunSucceeded || run.State == RunFailed || run.State == RunPolicyBlocked {
		out.MayAdvance = false
		out.PolicyDecision = PolicyEvalResult{Decision: DecisionDeny, Reason: "run_terminal"}
		return out, nil
	}

	// Identity binding
	if req.ProjectID != "" && run.ProjectID != req.ProjectID {
		out.PolicyDecision = PolicyEvalResult{Decision: DecisionDeny, Reason: "project_mismatch"}
		return out, nil
	}
	if req.TenantID != "" && run.TenantID != "" && run.TenantID != req.TenantID {
		out.PolicyDecision = PolicyEvalResult{Decision: DecisionDeny, Reason: "tenant_mismatch"}
		return out, nil
	}
	if req.ScopeID != "" && run.ScopeID != "" && run.ScopeID != req.ScopeID {
		out.PolicyDecision = PolicyEvalResult{Decision: DecisionDeny, Reason: "scope_mismatch"}
		return out, nil
	}

	// Gate 1: autonomy policy
	cap := contracts.CapabilityID("")
	if run.Definition != nil && len(run.Definition.Steps) > 0 {
		// pick first incomplete-ish step id for policy check; Orchestrator owns ready-step selection
		cap = run.Definition.Steps[0].CapabilityID
	}
	dec, err := EvaluateAutonomyPolicy(PolicyEvalRequest{
		ProjectID: run.ProjectID, TenantID: run.TenantID, ScopeID: run.ScopeID,
		PrincipalID: run.PrincipalID, CorrelationID: run.CorrelationID,
		CapabilityID: cap, StepsExecuted: run.StepsConsumed,
		RuntimeElapsed: h.Controller.now().Sub(run.StartedAt),
		Policy:         run.Policy, LLMCallsUsed: run.LLMCallsUsed, NextAllowedAt: run.NextAllowedAt,
		NoProgressCount: run.NoProgressCount, Now: h.Controller.now(),
	})
	out.PolicyDecision = dec
	if err != nil {
		return out, err
	}
	if !MayAutoAdvance(dec) {
		out.MayAdvance = false
		if dec.Decision == DecisionWait {
			run.State = RunWaiting
			h.Controller.persist(run)
		}
		if dec.Decision == DecisionDeny || dec.Decision == DecisionStop {
			run.State = RunPolicyBlocked
			run.StopReason = dec.Reason
			h.Controller.persist(run)
		}
		return out, nil
	}

	// Gate 2: Orchestrator (fresh CP → Grant inside)
	if h.Orchestrator == nil {
		return out, fmt.Errorf("%w: orchestrator not configured", contracts.ErrInvalidArgument)
	}
	if run.Definition == nil {
		return out, fmt.Errorf("%w: workflow definition missing", contracts.ErrInvalidArgument)
	}

	wfRunID := req.WorkflowRunID
	if wfRunID == "" {
		// Start workflow run once if needed
		started, err := h.Orchestrator.Start(ctx, workflow.StartRequest{
			Definition:    *run.Definition,
			PrincipalID:   run.PrincipalID,
			CorrelationID: run.CorrelationID,
		})
		if err != nil {
			out.Err = err
			return out, err
		}
		wfRunID = started.ID
		run.WorkflowDefID = run.Definition.ID
		// store workflow run id in stop_reason field is wrong - use LastDecision metadata
		run.LastDecision = "workflow_started:" + wfRunID
		h.Controller.persist(run)
	}

	out.OrchestratorCalled = true
	wfRun, err := h.Orchestrator.Advance(ctx, wfRunID, *run.Definition)
	out.WorkflowRun = wfRun
	if err != nil {
		out.Err = err
		// CP denial or lifecycle failure — not a Grant from autonomy
		h.Controller.persist(run)
		return out, err
	}

	// Budget accounting after authoritative advance attempt
	_ = h.Controller.RecordStepExecuted(run.ID)
	run, _ = h.Controller.Get(req.RunID)
	out.Run = run
	out.MayAdvance = true
	return out, nil
}

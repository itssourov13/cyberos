package autonomy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/aicontext"
	"cyberos/internal/ai/apply"
	"cyberos/internal/contracts"
	"cyberos/internal/workflow"
)

// ProcessStatus describes EventAutonomyProcessor outcome (no execution).
type ProcessStatus string

const (
	ProcessOK             ProcessStatus = "OK"
	ProcessRejected       ProcessStatus = "REJECTED"
	ProcessStale          ProcessStatus = "STALE_CONTEXT"
	ProcessTerminal       ProcessStatus = "TERMINAL"
	ProcessPolicyWait     ProcessStatus = "POLICY_WAIT"
	ProcessPolicyDeny     ProcessStatus = "POLICY_DENY"
	ProcessPolicyEligible ProcessStatus = "POLICY_ELIGIBLE"
	ProcessIdempotentSkip ProcessStatus = "IDEMPOTENT_SKIP"
)

// ProcessResult is the structured outcome of event-driven observe→analyze→plan→policy.
// Never contains grants, secrets, or execution results from Broker.
type ProcessResult struct {
	EventID              string
	IdempotencyKey       string
	ObservationEventID   string
	AnalysisID           string
	PlanID               string
	WorkflowDefinitionID string
	PolicyDecision       PolicyEvalResult
	AdvanceEligible      bool
	Status               ProcessStatus
	Run                  *AutonomyRun
	Definition           *workflow.Definition
	Lineage              PlanLineage
	Error                string
}

// EventAutonomyProcessor connects 15A events to Analyze → RePlan → Policy (eligibility only).
// It never calls Broker, GrantIssuer, Sandbox, or Executor. It does not invoke AdvanceHandoff.Execute
// (which would reach Orchestrator); it only reports eligibility for a later explicit handoff.
type EventAutonomyProcessor struct {
	Controller *Controller
	Analyzer   *Analyzer
	RePlanner  *RePlanner
	Assembler  *aicontext.Assembler

	mu   sync.Mutex
	seen map[string]struct{} // project|idempotency_key — in-process dedupe only
	Now  func() time.Time
}

// NewEventAutonomyProcessor wires processor over existing autonomy/AI stack.
func NewEventAutonomyProcessor(ctrl *Controller, brain ai.Brain) *EventAutonomyProcessor {
	if brain == nil {
		brain = &ai.MockBrain{}
	}
	asm := aicontext.NewAssembler()
	return &EventAutonomyProcessor{
		Controller: ctrl,
		Analyzer:   NewAnalyzer(brain, asm),
		RePlanner:  NewRePlanner(brain, apply.NewApplicator(nil)),
		Assembler:  asm,
		seen:       make(map[string]struct{}),
		Now:        time.Now,
	}
}

// Process runs the bounded chain: Event → Observation → Analyze → Plan → Validate → Apply → Policy → STOP.
func (p *EventAutonomyProcessor) Process(ctx context.Context, event AutonomyEvent) (*ProcessResult, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: processor not configured", contracts.ErrInvalidArgument)
	}
	out := &ProcessResult{EventID: event.EventID, IdempotencyKey: event.IdempotencyKey}

	if err := ValidateAutonomyEvent(event); err != nil {
		out.Status = ProcessRejected
		out.Error = err.Error()
		return out, err
	}

	obs, err := ToEventObservation(event)
	if err != nil {
		out.Status = ProcessRejected
		out.Error = err.Error()
		return out, err
	}
	out.ObservationEventID = obs.EventID

	// In-process idempotency (15C owns durable outbox).
	idKey := event.IdempotencyIdentity()
	p.mu.Lock()
	if _, ok := p.seen[idKey]; ok {
		p.mu.Unlock()
		out.Status = ProcessIdempotentSkip
		return out, nil
	}
	p.seen[idKey] = struct{}{}
	p.mu.Unlock()

	var run *AutonomyRun
	if event.AutonomyRunID != "" && p.Controller != nil {
		run, err = p.Controller.Get(event.AutonomyRunID)
		if err != nil && p.Controller.Repo != nil && event.ProjectID != "" {
			run, err = p.Controller.LoadFromRepo(ctx, event.ProjectID, event.AutonomyRunID)
		}
		if err == nil && run != nil {
			out.Run = run
			// Terminal / cancelled: no progression toward eligibility path that could later execute.
			switch run.State {
			case RunCancelled, RunSucceeded, RunFailed, RunPolicyBlocked:
				out.Status = ProcessTerminal
				out.Error = "run_terminal"
				return out, nil
			}
			// Stale / identity binding against authoritative run.
			if run.ProjectID != "" && run.ProjectID != event.ProjectID {
				out.Status = ProcessStale
				out.Error = "project_mismatch"
				return out, nil
			}
			if run.TenantID != "" && event.TenantID != "" && run.TenantID != event.TenantID {
				out.Status = ProcessStale
				out.Error = "tenant_mismatch"
				return out, nil
			}
			if run.ScopeID != "" && event.ScopeID != "" && run.ScopeID != event.ScopeID {
				out.Status = ProcessStale
				out.Error = "scope_mismatch"
				return out, nil
			}
		}
	}

	// Observation → Analyze (Phase 13 ObservationRequest shape).
	goal := "event-driven analysis"
	if run != nil && run.Goal != "" {
		goal = run.Goal
	}
	anReq := ObservationRequest{
		ProjectID: event.ProjectID, TenantID: event.TenantID, ScopeID: event.ScopeID,
		CorrelationID: event.CorrelationID, Objective: goal,
		SourceRef: "event:" + event.EventID,
	}
	if run != nil {
		anReq.PrincipalID = run.PrincipalID
	}
	analysis, err := p.Analyzer.AnalyzeObservations(ctx, anReq)
	if err != nil {
		out.Status = ProcessRejected
		out.Error = err.Error()
		return out, err
	}
	out.AnalysisID = analysis.AnalysisID

	// Analysis → RePlan → ValidatePlan → Applicator (definition only).
	rpReq := RePlanRequest{
		ProjectID: event.ProjectID, TenantID: event.TenantID, ScopeID: event.ScopeID,
		CorrelationID: event.CorrelationID, PreviousAnalysisID: analysis.AnalysisID,
		Goal: goal, Analysis: analysis, AcceptApply: true,
	}
	if run != nil {
		rpReq.PrincipalID = run.PrincipalID
		rpReq.PreviousPlanID = run.CurrentPlanID
		if len(run.CapabilityHints) > 0 {
			rpReq.CapabilityHints = run.CapabilityHints
		}
	}
	rpRes, err := p.RePlanner.RePlan(ctx, rpReq)
	if err != nil {
		out.Status = ProcessRejected
		out.Error = err.Error()
		return out, err
	}
	out.PlanID = rpRes.Lineage.PlanID
	out.WorkflowDefinitionID = rpRes.Lineage.WorkflowDefinitionID
	out.Lineage = rpRes.Lineage
	out.Definition = rpRes.Definition

	// Policy eligibility (no Handoff.Execute → no Orchestrator in 15B).
	policy := AutonomyPolicy{
		Version: PolicyContractV1, Mode: ModeManual, ProjectID: event.ProjectID, MaxSteps: 1,
	}
	if run != nil {
		policy = run.Policy
	}
	cap := contracts.CapabilityID("")
	if rpRes.Definition != nil && len(rpRes.Definition.Steps) > 0 {
		cap = rpRes.Definition.Steps[0].CapabilityID
	}
	steps := 0
	started := time.Time{}
	if run != nil {
		steps = run.StepsConsumed
		started = run.StartedAt
	}
	elapsed := time.Duration(0)
	if !started.IsZero() && p.Now != nil {
		elapsed = p.Now().Sub(started)
	}
	dec, err := EvaluateAutonomyPolicy(PolicyEvalRequest{
		ProjectID: event.ProjectID, TenantID: event.TenantID, ScopeID: event.ScopeID,
		CapabilityID: cap, StepsExecuted: steps, RuntimeElapsed: elapsed, Policy: policy,
	})
	out.PolicyDecision = dec
	if err != nil {
		out.Status = ProcessRejected
		out.Error = err.Error()
		return out, err
	}

	switch {
	case MayAutoAdvance(dec):
		out.AdvanceEligible = true
		out.Status = ProcessPolicyEligible
		// STOP — caller may later invoke AdvanceHandoff explicitly; 15B does not.
	case dec.Decision == DecisionWait:
		out.Status = ProcessPolicyWait
	default:
		out.Status = ProcessPolicyDeny
	}

	// Persist definition reference on run if present (no grant, no advance).
	if run != nil && p.Controller != nil && rpRes.Definition != nil {
		run.Definition = rpRes.Definition
		run.CurrentPlanID = rpRes.Lineage.PlanID
		run.AnalysisID = analysis.AnalysisID
		run.ParentPlanID = rpRes.Lineage.ParentPlanID
		run.WorkflowDefID = rpRes.Lineage.WorkflowDefinitionID
		run.UpdatedAt = p.Now()
		p.Controller.persist(run)
		out.Run = run
	}

	return out, nil
}

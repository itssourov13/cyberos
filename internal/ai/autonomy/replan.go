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

// RePlanRequest requests a new capability-first plan from an analysis (Phase 13B).
// Authoritative project/tenant/scope come from the caller — not the model.
type RePlanRequest struct {
	ProjectID          contracts.ProjectID
	TenantID           contracts.TenantID
	ScopeID            contracts.ScopeID
	PrincipalID        contracts.PrincipalID
	CorrelationID      contracts.CorrelationID
	CausationID        contracts.CausationID
	PreviousAnalysisID string
	PreviousPlanID     string
	PreviousWorkflowID string
	Goal               string
	Analysis           *AnalysisResult // optional; if set, must match project
	CapabilityHints    []contracts.CapabilityID
	ProviderID         string
	Model              string
	// AcceptApply if true applies the validated plan into a Workflow Definition (not execution).
	AcceptApply bool
}

// PlanLineage links analysis → plan → workflow definition.
type PlanLineage struct {
	PlanID               string
	ParentPlanID         string
	AnalysisID           string
	ParentAnalysisID     string
	WorkflowDefinitionID string
	ProjectID            contracts.ProjectID
	TenantID             contracts.TenantID
	ScopeID              contracts.ScopeID
	PrincipalID          contracts.PrincipalID
	ProviderID           string
	Model                string
	CorrelationID        contracts.CorrelationID
	CausationID          contracts.CausationID
	CreatedAt            time.Time
}

// RePlanResult is the outcome of one re-plan operation (no execution).
type RePlanResult struct {
	Plan       *ai.Plan
	Lineage    PlanLineage
	Definition *workflow.Definition // set only if AcceptApply
	Applied    bool
	Idempotent bool
}

// LineageStore holds plan lineage (in-memory).
type LineageStore struct {
	mu   sync.RWMutex
	byID map[string]PlanLineage
}

// NewLineageStore constructs an empty lineage store.
func NewLineageStore() *LineageStore {
	return &LineageStore{byID: make(map[string]PlanLineage)}
}

func (s *LineageStore) Put(l PlanLineage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[l.PlanID] = l
}

func (s *LineageStore) Get(planID string) (PlanLineage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.byID[planID]
	return l, ok
}

// RePlanner runs Analyze-context → Brain.Plan → Validate → optional Apply.
type RePlanner struct {
	Brain      ai.Brain
	Assembler  *aicontext.Assembler
	Apply      *apply.Applicator
	Lineage    *LineageStore
	SourceData aicontext.SourceData
	Now        func() time.Time
}

// NewRePlanner constructs a re-planner.
func NewRePlanner(brain ai.Brain, app *apply.Applicator) *RePlanner {
	if brain == nil {
		brain = &ai.MockBrain{}
	}
	if app == nil {
		app = apply.NewApplicator(nil)
	}
	return &RePlanner{
		Brain: brain, Assembler: aicontext.NewAssembler(), Apply: app,
		Lineage: NewLineageStore(), Now: time.Now,
	}
}

// RePlan produces a validated plan from analysis/context; optionally applies to Definition.
func (r *RePlanner) RePlan(ctx context.Context, req RePlanRequest) (*RePlanResult, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: replanner not configured", contracts.ErrInvalidArgument)
	}
	if req.ProjectID == "" || req.ScopeID == "" {
		return nil, fmt.Errorf("%w: project_id and scope_id required", contracts.ErrInvalidArgument)
	}
	if req.Goal == "" {
		return nil, fmt.Errorf("%w: goal required", contracts.ErrInvalidArgument)
	}
	if req.Analysis != nil {
		if req.Analysis.ProjectID != "" && req.Analysis.ProjectID != req.ProjectID {
			return nil, fmt.Errorf("%w: analysis project mismatch", contracts.ErrForbidden)
		}
		if req.TenantID != "" && req.Analysis.TenantID != "" && req.Analysis.TenantID != req.TenantID {
			return nil, fmt.Errorf("%w: analysis tenant mismatch", contracts.ErrForbidden)
		}
		if req.PreviousAnalysisID == "" {
			req.PreviousAnalysisID = req.Analysis.AnalysisID
		}
	}

	// Assemble bounded context (isolation + redaction) — no secrets resolved.
	_, err := r.Assembler.Assemble(aicontext.ContextRequest{
		TenantID: req.TenantID, ProjectID: req.ProjectID, ScopeID: req.ScopeID,
		PrincipalID: req.PrincipalID, CorrelationID: req.CorrelationID,
		IncludeCapabilities: true, IncludeFindings: true, IncludeEvidence: true,
	}, r.SourceData)
	if err != nil {
		return nil, err
	}

	brainReq := ai.BrainRequest{
		Operation: ai.OpPlan, PrincipalID: req.PrincipalID, ProjectID: req.ProjectID,
		ScopeID: req.ScopeID, TenantID: req.TenantID, Objective: req.Goal,
		CorrelationID: req.CorrelationID, CapabilityHints: req.CapabilityHints,
	}
	resp, err := r.Brain.Plan(ctx, brainReq)
	if err != nil {
		return nil, err
	}
	if resp.Status == "error" || resp.Plan == nil {
		return nil, fmt.Errorf("%w: brain plan failed: %s", contracts.ErrInvalidArgument, resp.Error)
	}

	plan := *resp.Plan
	// Authoritative binding — model cannot rewrite isolation.
	plan.ProjectID = req.ProjectID
	plan.ScopeID = req.ScopeID
	plan.TenantID = req.TenantID
	plan.PrincipalID = req.PrincipalID
	if plan.CorrelationID == "" {
		plan.CorrelationID = req.CorrelationID
	}
	if plan.CausationID == "" {
		plan.CausationID = req.CausationID
	}
	if err := ai.ValidatePlan(&plan); err != nil {
		return nil, err
	}
	if len(plan.Steps) > 50 {
		return nil, fmt.Errorf("%w: too many plan steps", contracts.ErrInvalidArgument)
	}

	now := time.Now()
	if r.Now != nil {
		now = r.Now()
	}
	lineage := PlanLineage{
		PlanID: plan.ID, ParentPlanID: req.PreviousPlanID,
		AnalysisID: req.PreviousAnalysisID, ParentAnalysisID: req.PreviousAnalysisID,
		ProjectID: req.ProjectID, TenantID: req.TenantID, ScopeID: req.ScopeID,
		PrincipalID: req.PrincipalID, ProviderID: req.ProviderID, Model: req.Model,
		CorrelationID: req.CorrelationID, CausationID: req.CausationID, CreatedAt: now,
	}
	if resp.ProviderMeta != nil {
		if lineage.ProviderID == "" {
			lineage.ProviderID = resp.ProviderMeta["provider"]
		}
		if lineage.Model == "" {
			lineage.Model = resp.ProviderMeta["model"]
		}
	}

	out := &RePlanResult{Plan: &plan, Lineage: lineage}
	if !req.AcceptApply {
		r.Lineage.Put(lineage)
		return out, nil
	}

	applyRes, err := r.Apply.Apply(ctx, apply.ApplyRequest{
		Plan: plan, ProjectID: req.ProjectID, TenantID: req.TenantID, ScopeID: req.ScopeID,
		PrincipalID: req.PrincipalID, CorrelationID: req.CorrelationID, CausationID: req.CausationID,
		ProviderID: req.ProviderID, Model: req.Model, Accepted: true, // explicit AcceptApply
		SessionID: "replan-" + plan.ID,
	})
	if err != nil {
		return nil, err
	}
	lineage.WorkflowDefinitionID = applyRes.WorkflowID
	r.Lineage.Put(lineage)
	out.Lineage = lineage
	out.Definition = &applyRes.Definition
	out.Applied = true
	out.Idempotent = applyRes.Idempotent
	return out, nil
}

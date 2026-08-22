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

// RunState is the AutonomyRun state machine.
type RunState string

const (
	RunPending       RunState = "PENDING"
	RunRunning       RunState = "RUNNING"
	RunWaiting       RunState = "WAITING"
	RunSucceeded     RunState = "SUCCEEDED"
	RunFailed        RunState = "FAILED"
	RunCancelled     RunState = "CANCELLED"
	RunPolicyBlocked RunState = "POLICY_BLOCKED"
)

// AutonomyRun is one continuous controlled autonomous operation (Phase 13D).
// It does not issue grants or call Broker; advancement is eligibility-only.
type AutonomyRun struct {
	ID              string                   `json:"id"`
	Goal            string                   `json:"goal"`
	ProjectID       contracts.ProjectID      `json:"project_id"`
	TenantID        contracts.TenantID       `json:"tenant_id,omitempty"`
	ScopeID         contracts.ScopeID        `json:"scope_id"`
	PrincipalID     contracts.PrincipalID    `json:"principal_id,omitempty"`
	CorrelationID   contracts.CorrelationID  `json:"correlation_id,omitempty"`
	Policy          AutonomyPolicy           `json:"policy"`
	State           RunState                 `json:"state"`
	Iteration       int                      `json:"iteration"`
	StepsConsumed   int                      `json:"steps_consumed"`
	CurrentPlanID   string                   `json:"current_plan_id,omitempty"`
	ParentPlanID    string                   `json:"parent_plan_id,omitempty"`
	AnalysisID      string                   `json:"analysis_id,omitempty"`
	WorkflowDefID   string                   `json:"workflow_definition_id,omitempty"`
	LastDecision    string                   `json:"last_decision,omitempty"`
	StopReason      string                   `json:"stop_reason,omitempty"`
	StartedAt       time.Time                `json:"started_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
	FinishedAt      time.Time                `json:"finished_at,omitempty"`
	Definition      *workflow.Definition     `json:"-"`
	CapabilityHints []contracts.CapabilityID `json:"capability_hints,omitempty"`
}

// RunStore is in-memory durable-within-process store for autonomy runs.
type RunStore struct {
	mu   sync.RWMutex
	runs map[string]*AutonomyRun
}

// NewRunStore constructs an empty run store.
func NewRunStore() *RunStore {
	return &RunStore{runs: make(map[string]*AutonomyRun)}
}

func (s *RunStore) Put(r *AutonomyRun) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *r
	s.runs[r.ID] = &cp
}

func (s *RunStore) Get(id string) (*AutonomyRun, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	if !ok {
		return nil, false
	}
	cp := *r
	return &cp, true
}

// Controller orchestrates Observe→Analyze→Plan→Policy (no Broker).
type Controller struct {
	Store     *RunStore
	Analyzer  *Analyzer
	RePlanner *RePlanner
	Now       func() time.Time
}

// NewController constructs an autonomy controller.
func NewController(brain ai.Brain) *Controller {
	app := apply.NewApplicator(nil)
	return &Controller{
		Store:     NewRunStore(),
		Analyzer:  NewAnalyzer(brain, aicontext.NewAssembler()),
		RePlanner: NewRePlanner(brain, app),
		Now:       time.Now,
	}
}

// StartRequest begins an autonomy run (identity authoritative).
type StartRequest struct {
	ID              string
	Goal            string
	ProjectID       contracts.ProjectID
	TenantID        contracts.TenantID
	ScopeID         contracts.ScopeID
	PrincipalID     contracts.PrincipalID
	CorrelationID   contracts.CorrelationID
	Policy          AutonomyPolicy
	CapabilityHints []contracts.CapabilityID
	SourceData      aicontext.SourceData
}

// Start creates a PENDING→RUNNING run and performs first observe/analyze/plan/policy cycle.
// Does not call Orchestrator.Advance or issue grants.
func (c *Controller) Start(ctx context.Context, req StartRequest) (*AutonomyRun, error) {
	if req.ProjectID == "" || req.ScopeID == "" || req.Goal == "" {
		return nil, fmt.Errorf("%w: project, scope, goal required", contracts.ErrInvalidArgument)
	}
	req.Policy.ProjectID = req.ProjectID
	if req.Policy.TenantID == "" {
		req.Policy.TenantID = req.TenantID
	}
	if req.Policy.ScopeID == "" {
		req.Policy.ScopeID = req.ScopeID
	}
	if err := ValidatePolicy(req.Policy); err != nil {
		return nil, err
	}
	id := req.ID
	if id == "" {
		id = fmt.Sprintf("arun-%d", c.now().UnixNano())
	}
	if existing, ok := c.Store.Get(id); ok {
		return existing, nil // idempotent start
	}
	run := &AutonomyRun{
		ID: id, Goal: req.Goal, ProjectID: req.ProjectID, TenantID: req.TenantID,
		ScopeID: req.ScopeID, PrincipalID: req.PrincipalID, CorrelationID: req.CorrelationID,
		Policy: req.Policy, State: RunPending, CapabilityHints: req.CapabilityHints,
		StartedAt: c.now(), UpdatedAt: c.now(),
	}
	c.Analyzer.SourceData = req.SourceData
	c.RePlanner.SourceData = req.SourceData
	if err := c.transition(run, RunRunning); err != nil {
		return nil, err
	}
	if err := c.iterate(ctx, run); err != nil {
		run.State = RunFailed
		run.StopReason = err.Error()
		run.FinishedAt = c.now()
		c.Store.Put(run)
		return run, err
	}
	c.Store.Put(run)
	return run, nil
}

// Advance continues a controlled autonomous run by one iteration (policy-gated).
// Does not call Broker; returns eligibility for external Orchestrator.Advance.
type AdvanceResult struct {
	Run            *AutonomyRun
	PolicyDecision PolicyEvalResult
	MayAdvance     bool
	Definition     *workflow.Definition
}

// Advance re-evaluates policy and optionally prepares next plan iteration.
func (c *Controller) Advance(ctx context.Context, runID string) (*AdvanceResult, error) {
	run, ok := c.Store.Get(runID)
	if !ok {
		return nil, contracts.ErrNotFound
	}
	if run.State == RunCancelled || run.State == RunSucceeded || run.State == RunFailed || run.State == RunPolicyBlocked {
		return &AdvanceResult{Run: run, MayAdvance: false}, nil
	}
	if run.State == RunWaiting {
		// resume path: re-evaluate only
	} else if run.State != RunRunning {
		return nil, fmt.Errorf("%w: cannot advance from %s", contracts.ErrInvalidArgument, run.State)
	}

	// Budget check before iteration
	dec, err := c.evalPolicy(run, "")
	if err != nil {
		return nil, err
	}
	if dec.Decision == DecisionStop || dec.Decision == DecisionDeny {
		run.State = RunPolicyBlocked
		run.StopReason = dec.Reason
		run.LastDecision = string(dec.Decision)
		run.FinishedAt = c.now()
		run.UpdatedAt = c.now()
		c.Store.Put(run)
		return &AdvanceResult{Run: run, PolicyDecision: dec, MayAdvance: false}, nil
	}
	if dec.Decision == DecisionWait {
		run.State = RunWaiting
		run.LastDecision = string(dec.Decision)
		run.UpdatedAt = c.now()
		c.Store.Put(run)
		return &AdvanceResult{Run: run, PolicyDecision: dec, MayAdvance: false, Definition: run.Definition}, nil
	}

	// ALLOW under CONTROLLED_AUTONOMOUS — iterate plan if needed
	if err := c.iterate(ctx, run); err != nil {
		run.State = RunFailed
		run.StopReason = err.Error()
		run.FinishedAt = c.now()
		c.Store.Put(run)
		return &AdvanceResult{Run: run, MayAdvance: false}, err
	}
	// Re-check policy for capability on current plan
	cap := contracts.CapabilityID("")
	if run.Definition != nil && len(run.Definition.Steps) > 0 {
		cap = run.Definition.Steps[0].CapabilityID
	}
	dec, _ = c.evalPolicy(run, cap)
	run.LastDecision = string(dec.Decision)
	may := MayAutoAdvance(dec) && run.State == RunRunning
	if !may && (dec.Decision == DecisionWait) {
		run.State = RunWaiting
	}
	if dec.Decision == DecisionDeny || dec.Decision == DecisionStop {
		run.State = RunPolicyBlocked
		run.StopReason = dec.Reason
		run.FinishedAt = c.now()
		may = false
	}
	run.UpdatedAt = c.now()
	c.Store.Put(run)
	return &AdvanceResult{Run: run, PolicyDecision: dec, MayAdvance: may, Definition: run.Definition}, nil
}

// Cancel stops further autonomous progression (idempotent).
func (c *Controller) Cancel(runID string) (*AutonomyRun, error) {
	run, ok := c.Store.Get(runID)
	if !ok {
		return nil, contracts.ErrNotFound
	}
	if run.State == RunCancelled {
		return run, nil
	}
	if run.State == RunSucceeded || run.State == RunFailed {
		return run, nil
	}
	run.State = RunCancelled
	run.StopReason = "operator_cancelled"
	run.FinishedAt = c.now()
	run.UpdatedAt = c.now()
	c.Store.Put(run)
	return run, nil
}

// Get returns a copy of the run.
func (c *Controller) Get(runID string) (*AutonomyRun, error) {
	run, ok := c.Store.Get(runID)
	if !ok {
		return nil, contracts.ErrNotFound
	}
	return run, nil
}

// RecordStepExecuted increments step budget after external Orchestrator execution.
// Caller is responsible for having obtained a Grant via CP.
func (c *Controller) RecordStepExecuted(runID string) error {
	run, ok := c.Store.Get(runID)
	if !ok {
		return contracts.ErrNotFound
	}
	run.StepsConsumed++
	run.UpdatedAt = c.now()
	c.Store.Put(run)
	return nil
}

func (c *Controller) iterate(ctx context.Context, run *AutonomyRun) error {
	run.Iteration++
	// Observe → Analyze
	ar, err := c.Analyzer.AnalyzeObservations(ctx, ObservationRequest{
		ProjectID: run.ProjectID, TenantID: run.TenantID, ScopeID: run.ScopeID,
		PrincipalID: run.PrincipalID, CorrelationID: run.CorrelationID,
		Objective: run.Goal, SourceRef: run.ID,
	})
	if err != nil {
		return err
	}
	run.AnalysisID = ar.AnalysisID

	// Re-plan → Validate → Apply
	accept := run.Policy.Mode == ModeControlledAutonomous || run.Policy.Mode == ModeAssisted || run.Policy.Mode == ModeManual
	// Always produce definition for inspection; auto-advance still gated by policy
	rp, err := c.RePlanner.RePlan(ctx, RePlanRequest{
		ProjectID: run.ProjectID, TenantID: run.TenantID, ScopeID: run.ScopeID,
		PrincipalID: run.PrincipalID, CorrelationID: run.CorrelationID,
		PreviousAnalysisID: ar.AnalysisID, PreviousPlanID: run.CurrentPlanID,
		PreviousWorkflowID: run.WorkflowDefID, Goal: run.Goal, Analysis: ar,
		CapabilityHints: run.CapabilityHints, AcceptApply: accept,
	})
	if err != nil {
		return err
	}
	run.ParentPlanID = run.CurrentPlanID
	run.CurrentPlanID = rp.Plan.ID
	if rp.Definition != nil {
		run.Definition = rp.Definition
		run.WorkflowDefID = rp.Definition.ID
	}
	run.UpdatedAt = c.now()
	return nil
}

func (c *Controller) evalPolicy(run *AutonomyRun, cap contracts.CapabilityID) (PolicyEvalResult, error) {
	elapsed := c.now().Sub(run.StartedAt)
	return EvaluateAutonomyPolicy(PolicyEvalRequest{
		ProjectID: run.ProjectID, TenantID: run.TenantID, ScopeID: run.ScopeID,
		PrincipalID: run.PrincipalID, CorrelationID: run.CorrelationID,
		CapabilityID: cap, StepsExecuted: run.StepsConsumed, RuntimeElapsed: elapsed,
		Policy: run.Policy,
	})
}

func (c *Controller) transition(run *AutonomyRun, to RunState) error {
	from := run.State
	ok := false
	switch from {
	case RunPending:
		ok = to == RunRunning
	case RunRunning:
		ok = to == RunWaiting || to == RunSucceeded || to == RunFailed || to == RunCancelled || to == RunPolicyBlocked || to == RunRunning
	case RunWaiting:
		ok = to == RunRunning || to == RunCancelled || to == RunWaiting
	}
	if from == "" && to == RunPending {
		ok = true
	}
	if !ok && from != to {
		// allow initial
		if from == RunPending && to == RunRunning {
			ok = true
		}
	}
	if !ok && from != RunPending {
		// permissive for start path
		if from == RunPending {
			ok = true
		}
	}
	run.State = to
	run.UpdatedAt = c.now()
	return nil
}

func (c *Controller) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

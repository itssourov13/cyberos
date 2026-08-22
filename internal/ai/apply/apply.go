// Package apply converts validated AI Plans into Workflow Definitions (Phase 12D).
// Apply does not authorize, issue grants, call Broker, or execute tools.
package apply

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/contracts"
	"cyberos/internal/workflow"
)

// ApplyRequest binds a plan to an authoritative caller context.
type ApplyRequest struct {
	Plan          ai.Plan
	ProjectID     contracts.ProjectID
	TenantID      contracts.TenantID
	ScopeID       contracts.ScopeID
	PrincipalID   contracts.PrincipalID
	CorrelationID contracts.CorrelationID
	CausationID   contracts.CausationID
	SessionID     string // optional AI session
	ProviderID    string
	Model         string
	// Accepted must be true — applying without explicit acceptance is rejected.
	Accepted bool
}

// ApplyResult is the outcome of plan application (workflow created, not executed).
type ApplyResult struct {
	SessionID  string
	PlanID     string
	WorkflowID string
	Definition workflow.Definition
	Status     string // WORKFLOW_CREATED
	Idempotent bool
}

// Session is durable metadata for an AI planning interaction (no secrets/prompts).
type Session struct {
	ID            string
	ProviderID    string
	Model         string
	ProjectID     contracts.ProjectID
	TenantID      contracts.TenantID
	PrincipalID   contracts.PrincipalID
	CorrelationID contracts.CorrelationID
	CreatedAt     time.Time
}

// PlanRecord links a plan to a workflow after acceptance.
type PlanRecord struct {
	PlanID     string
	SessionID  string
	WorkflowID string
	Status     string // ACCEPTED | WORKFLOW_CREATED
	CreatedAt  time.Time
	AcceptedAt time.Time
}

// Store is an in-memory provenance store (restart-local; PG optional later).
type Store struct {
	mu       sync.RWMutex
	sessions map[string]Session
	plans    map[string]PlanRecord  // key: sessionID+"|"+planID or planID
	applied  map[string]ApplyResult // idempotency: planID+project
}

// NewStore constructs an empty provenance store.
func NewStore() *Store {
	return &Store{
		sessions: make(map[string]Session),
		plans:    make(map[string]PlanRecord),
		applied:  make(map[string]ApplyResult),
	}
}

func (s *Store) putSession(sess Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess.ID] = sess
}

func (s *Store) putPlan(rec PlanRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := rec.SessionID + "|" + rec.PlanID
	s.plans[key] = rec
	s.plans[rec.PlanID] = rec
}

func (s *Store) getApplied(key string) (ApplyResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.applied[key]
	return r, ok
}

func (s *Store) setApplied(key string, r ApplyResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applied[key] = r
}

// GetPlanRecord returns provenance for a plan id.
func (s *Store) GetPlanRecord(planID string) (PlanRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.plans[planID]
	return r, ok
}

// Applicator converts accepted plans into workflow definitions.
type Applicator struct {
	Store *Store
	Now   func() time.Time
}

// NewApplicator constructs an applicator with in-memory provenance.
func NewApplicator(store *Store) *Applicator {
	if store == nil {
		store = NewStore()
	}
	return &Applicator{Store: store, Now: time.Now}
}

// Apply validates the plan against the request context and creates a Workflow Definition.
// It does not start the workflow, issue grants, or execute tools.
func (a *Applicator) Apply(ctx context.Context, req ApplyRequest) (*ApplyResult, error) {
	if a == nil {
		return nil, fmt.Errorf("%w: applicator not configured", contracts.ErrInvalidArgument)
	}
	if !req.Accepted {
		return nil, fmt.Errorf("%w: plan must be explicitly accepted before apply", contracts.ErrForbidden)
	}
	if err := ai.ValidatePlan(&req.Plan); err != nil {
		return nil, err
	}
	// Authoritative context binding — plan cannot rewrite project/scope/tenant.
	if req.ProjectID == "" || req.ScopeID == "" {
		return nil, fmt.Errorf("%w: project_id and scope_id required", contracts.ErrInvalidArgument)
	}
	if req.Plan.ProjectID != "" && req.Plan.ProjectID != req.ProjectID {
		return nil, fmt.Errorf("%w: plan project_id mismatch", contracts.ErrForbidden)
	}
	if req.Plan.ScopeID != "" && req.Plan.ScopeID != req.ScopeID {
		return nil, fmt.Errorf("%w: plan scope_id mismatch", contracts.ErrForbidden)
	}
	if req.TenantID != "" && req.Plan.TenantID != "" && req.Plan.TenantID != req.TenantID {
		return nil, fmt.Errorf("%w: plan tenant_id mismatch", contracts.ErrForbidden)
	}

	idemKey := string(req.ProjectID) + "|" + req.Plan.ID
	if prev, ok := a.Store.getApplied(idemKey); ok {
		prev.Idempotent = true
		return &prev, nil
	}

	def, err := PlanToDefinition(req)
	if err != nil {
		return nil, err
	}
	if err := def.Validate(); err != nil {
		return nil, err
	}

	now := a.Now()
	if a.Now == nil {
		now = time.Now()
	}
	sessID := req.SessionID
	if sessID == "" {
		sessID = "ais-" + req.Plan.ID
	}
	a.Store.putSession(Session{
		ID: sessID, ProviderID: req.ProviderID, Model: req.Model,
		ProjectID: req.ProjectID, TenantID: req.TenantID, PrincipalID: req.PrincipalID,
		CorrelationID: req.CorrelationID, CreatedAt: now,
	})
	a.Store.putPlan(PlanRecord{
		PlanID: req.Plan.ID, SessionID: sessID, WorkflowID: def.ID,
		Status: "WORKFLOW_CREATED", CreatedAt: now, AcceptedAt: now,
	})

	res := ApplyResult{
		SessionID: sessID, PlanID: req.Plan.ID, WorkflowID: def.ID,
		Definition: def, Status: "WORKFLOW_CREATED",
	}
	a.Store.setApplied(idemKey, res)
	return &res, nil
}

// PlanToDefinition maps a plan to workflow.Definition (capability-first).
func PlanToDefinition(req ApplyRequest) (workflow.Definition, error) {
	p := req.Plan
	steps := make([]workflow.Step, 0, len(p.Steps))
	for _, s := range p.Steps {
		params := map[string]any{}
		for k, v := range s.Parameters {
			params[k] = v
		}
		steps = append(steps, workflow.Step{
			ID: s.ID, CapabilityID: s.CapabilityID, Parameters: params,
			DependsOn: append([]string{}, s.DependsOn...), Title: s.Rationale,
		})
	}
	name := p.Goal
	if name == "" {
		name = p.ID
	}
	return workflow.Definition{
		ID:        "wf-from-" + p.ID,
		Name:      name,
		Version:   "1.0.0",
		ProjectID: req.ProjectID,
		ScopeID:   req.ScopeID,
		Steps:     steps,
	}, nil
}

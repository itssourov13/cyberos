package assessment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/events"
	"cyberos/internal/evidence"
	"cyberos/internal/findings"
	"cyberos/internal/provenance"
)

// GrantIssuer issues a per-step ExecutionGrant after Control Plane evaluation.
// Orchestration never bypasses this — each tool step needs its own grant.
type GrantIssuer interface {
	IssueForStep(ctx context.Context, a *Assessment, step contracts.AssessmentStep) (contracts.ExecutionGrant, error)
}

// ExecutorGateway is the Tool Broker (or compatible) execution surface.
type ExecutorGateway interface {
	Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error)
}

// Orchestrator runs sequential assessment steps under independent grants.
type Orchestrator struct {
	issuer   GrantIssuer
	broker   ExecutorGateway
	evidence *evidence.Service
	findings *findings.Service
	prov     *provenance.Service
	events   *events.Store

	mu    sync.RWMutex
	items map[contracts.AssessmentID]*Assessment
}

// NewOrchestrator constructs an assessment orchestrator.
func NewOrchestrator(issuer GrantIssuer, broker ExecutorGateway) *Orchestrator {
	return &Orchestrator{
		issuer: issuer,
		broker: broker,
		items:  make(map[contracts.AssessmentID]*Assessment),
	}
}

// WithEvidence wires evidence/findings/provenance/events.
func (o *Orchestrator) WithEvidence(e *evidence.Service, f *findings.Service, p *provenance.Service, ev *events.Store) *Orchestrator {
	o.evidence = e
	o.findings = f
	o.prov = p
	o.events = ev
	return o
}

// Save stores an assessment.
func (o *Orchestrator) Save(a *Assessment) error {
	if a == nil {
		return contracts.ErrInvalidArgument
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	cp := *a
	cp.Steps = append([]contracts.AssessmentStep{}, a.Steps...)
	cp.StepResults = append([]StepResult{}, a.StepResults...)
	o.items[a.ID] = &cp
	return nil
}

// Get loads an assessment.
func (o *Orchestrator) Get(id contracts.AssessmentID) (*Assessment, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	a, ok := o.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *a
	return &cp, nil
}

// Run executes planned steps sequentially. Each step obtains its own grant.
func (o *Orchestrator) Run(ctx context.Context, id contracts.AssessmentID) (*Assessment, error) {
	if o == nil || o.issuer == nil || o.broker == nil {
		return nil, fmt.Errorf("%w: orchestrator not configured", contracts.ErrInvalidArgument)
	}
	a, err := o.Get(id)
	if err != nil {
		return nil, err
	}
	if a.State != contracts.AssessmentStatePlanned && a.State != contracts.AssessmentStateRunning {
		return nil, fmt.Errorf("%w: assessment must be PLANNED to run", contracts.ErrInvalidStateTransition)
	}
	if a.State == contracts.AssessmentStatePlanned {
		if err := a.TransitionTo(contracts.AssessmentStateRunning); err != nil {
			return nil, err
		}
	}
	o.emit(a, "assessment.started", nil)

	failed := false
	for _, step := range a.Steps {
		select {
		case <-ctx.Done():
			_ = a.Cancel()
			o.persist(a)
			o.emit(a, "assessment.cancelled", map[string]any{"reason": "context done"})
			return a, ctx.Err()
		default:
		}
		if a.Cancelled {
			break
		}
		sr := o.runStep(ctx, a, step)
		a.StepResults = append(a.StepResults, sr)
		if !sr.Success {
			failed = true
			// Safe failure: stop sequential chain; assessment marked failed.
			break
		}
	}

	if a.Cancelled {
		// already cancelled
	} else if failed {
		_ = a.TransitionTo(contracts.AssessmentStateFailed)
		o.emit(a, "assessment.failed", nil)
	} else {
		_ = a.TransitionTo(contracts.AssessmentStateSucceeded)
		o.emit(a, "assessment.succeeded", nil)
	}
	o.persist(a)
	return a, nil
}

func (o *Orchestrator) runStep(ctx context.Context, a *Assessment, step contracts.AssessmentStep) StepResult {
	now := time.Now().UTC()
	sr := StepResult{
		StepID:       step.ID,
		CapabilityID: step.CapabilityID,
		CompletedAt:  now,
	}
	grant, err := o.issuer.IssueForStep(ctx, a, step)
	if err != nil {
		sr.ErrorMessage = err.Error()
		sr.Summary = "grant issuance failed"
		o.emit(a, "assessment.step.denied", map[string]any{"step_id": step.ID, "error": err.Error()})
		return sr
	}
	sr.GrantID = grant.ID

	params := map[string]any{}
	for k, v := range step.Parameters {
		params[k] = v
	}
	if a.Target != "" {
		params["target"] = a.Target
	}
	res, err := o.broker.Execute(contracts.ExecutionRequest{
		Grant:         grant,
		Parameters:    params,
		CorrelationID: contracts.CorrelationID("assessment-" + string(a.ID) + "-" + step.ID),
	})
	if err != nil {
		sr.ErrorMessage = err.Error()
		sr.Summary = res.ErrorMessage
		if sr.Summary == "" {
			sr.Summary = "execution failed"
		}
		o.emit(a, "assessment.step.failed", map[string]any{"step_id": step.ID})
		return sr
	}
	sr.Success = res.Status == contracts.ExecutionStatusSucceeded
	if sum, ok := res.Output["summary"].(string); ok {
		sr.Summary = sum
	} else {
		sr.Summary = string(res.Status)
	}
	sr.ExecutionID = res.ExecutorID

	if o.evidence != nil && sr.Success {
		evID := contracts.EvidenceID(fmt.Sprintf("ev-%s-%s", a.ID, step.ID))
		ev, eerr := o.evidence.RecordFromExecution(res, grant, evID)
		if eerr == nil {
			sr.EvidenceID = ev.ID
			if o.prov != nil {
				_, _ = o.prov.RecordForEvidence(ev, res.ExecutorID)
			}
			if o.findings != nil {
				_, _ = o.findings.Create(findings.CreateRequest{
					ID:            contracts.FindingID(fmt.Sprintf("f-%s-%s", a.ID, step.ID)),
					TaskID:        grant.TaskID,
					ProjectID:     a.ProjectID,
					ScopeID:       a.ScopeID,
					Severity:      contracts.FindingSeverityInfo,
					Title:         "Assessment step: " + step.ID,
					EvidenceIDs:   []contracts.EvidenceID{ev.ID},
					CorrelationID: grant.CorrelationID,
				})
			}
		}
	}
	o.emit(a, "assessment.step.completed", map[string]any{
		"step_id": step.ID, "success": sr.Success, "grant_id": string(grant.ID),
	})
	return sr
}

func (o *Orchestrator) persist(a *Assessment) {
	_ = o.Save(a)
}

func (o *Orchestrator) emit(a *Assessment, typ string, payload map[string]any) {
	if o.events == nil {
		return
	}
	if payload == nil {
		payload = map[string]any{}
	}
	payload["assessment_id"] = string(a.ID)
	payload["state"] = string(a.State)
	_, _ = o.events.Append(events.AppendRequest{
		ID:            contracts.EventID(fmt.Sprintf("evt-%s-%d", typ, time.Now().UnixNano())),
		Type:          contracts.EventType(typ),
		ProjectID:     a.ProjectID,
		ScopeID:       a.ScopeID,
		CorrelationID: contracts.CorrelationID("assessment-" + string(a.ID)),
		Payload:       payload,
	})
}

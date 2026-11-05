package assessment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/events"
	"cyberos/internal/evidence"
	"cyberos/internal/findings"
	"cyberos/internal/provenance"
	"cyberos/internal/storage/ports"
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
	phase6   *Phase6Sink

	mu       sync.RWMutex
	items    map[contracts.AssessmentID]*Assessment
	repo     ports.AssessmentRepository
	runMu    sync.Mutex
	runLocks map[contracts.AssessmentID]*sync.Mutex
}

// NewOrchestrator constructs an assessment orchestrator.
func NewOrchestrator(issuer GrantIssuer, broker ExecutorGateway) *Orchestrator {
	return &Orchestrator{
		issuer:   issuer,
		broker:   broker,
		items:    make(map[contracts.AssessmentID]*Assessment),
		runLocks: make(map[contracts.AssessmentID]*sync.Mutex),
	}
}

// WithRepository attaches durable assessment state. Every state transition and
// completed step must persist before the caller observes success.
func (o *Orchestrator) WithRepository(repo ports.AssessmentRepository) *Orchestrator {
	if o != nil {
		o.repo = repo
	}
	return o
}

// WithEvidence wires evidence/findings/provenance/events.
func (o *Orchestrator) WithEvidence(e *evidence.Service, f *findings.Service, p *provenance.Service, ev *events.Store) *Orchestrator {
	o.evidence = e
	o.findings = f
	o.prov = p
	o.events = ev
	return o
}

// WithPhase6 wires the Phase 6 product pipeline (assets/findings/risk/cases). Never authorizes.
func (o *Orchestrator) WithPhase6(sink *Phase6Sink) *Orchestrator {
	o.phase6 = sink
	return o
}

// Save persists an assessment before publishing the new in-memory snapshot.
// Durable production state is never duplicated in an unbounded process cache.
func (o *Orchestrator) Save(a *Assessment) error {
	if o == nil || a == nil || a.ID == "" {
		return contracts.ErrInvalidArgument
	}
	cp := cloneAssessment(a)
	if o.repo != nil {
		rec, err := assessmentToRecord(cp)
		if err != nil {
			return err
		}
		current, getErr := o.repo.GetByID(context.Background(), string(cp.ID))
		switch {
		case getErr == nil:
			if current.ProjectID != string(cp.ProjectID) || current.ScopeID != string(cp.ScopeID) || current.PrincipalID != string(cp.PrincipalID) {
				return fmt.Errorf("%w: assessment ownership is immutable", contracts.ErrConflict)
			}
			if err := o.repo.Update(context.Background(), rec); err != nil {
				return fmt.Errorf("persist assessment update: %w", err)
			}
		case errors.Is(getErr, contracts.ErrNotFound):
			if err := o.repo.Create(context.Background(), rec); err != nil {
				return fmt.Errorf("persist assessment create: %w", err)
			}
		default:
			return fmt.Errorf("load assessment before save: %w", getErr)
		}
		return nil
	}
	o.mu.Lock()
	o.items[cp.ID] = cp
	o.mu.Unlock()
	return nil
}

// Get reads the durable source in production and a local snapshot in development.
func (o *Orchestrator) Get(id contracts.AssessmentID) (*Assessment, error) {
	if o == nil || id == "" {
		return nil, contracts.ErrInvalidArgument
	}
	if o.repo != nil {
		rec, err := o.repo.GetByID(context.Background(), string(id))
		if err != nil {
			return nil, err
		}
		return assessmentFromRecord(rec)
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	a, ok := o.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	return cloneAssessment(a), nil
}

// ListByProject returns durable assessments for one project, or the in-memory
// snapshot in development. It never materializes assessments across tenants.
func (o *Orchestrator) ListByProject(projectID contracts.ProjectID, limit, offset int) ([]*Assessment, error) {
	if o == nil || limit < 1 || limit > 500 || offset < 0 {
		return nil, contracts.ErrInvalidArgument
	}
	if o.repo != nil {
		var rows []ports.AssessmentRecord
		var err error
		if projectID == "" {
			rows, err = o.repo.List(context.Background(), limit, offset)
		} else {
			rows, err = o.repo.ListByProject(context.Background(), projectID, limit, offset)
		}
		if err != nil {
			return nil, err
		}
		out := make([]*Assessment, 0, len(rows))
		for _, row := range rows {
			if projectID != "" && row.ProjectID != string(projectID) {
				continue
			}
			a, err := assessmentFromRecord(row)
			if err != nil {
				return nil, err
			}
			out = append(out, a)
		}
		return out, nil
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	out := make([]*Assessment, 0)
	for _, a := range o.items {
		if a.ProjectID == projectID {
			out = append(out, cloneAssessment(a))
		}
	}
	return out, nil
}

func cloneAssessment(a *Assessment) *Assessment {
	if a == nil {
		return nil
	}
	cp := *a
	cp.Steps = append([]contracts.AssessmentStep(nil), a.Steps...)
	cp.StepResults = append([]StepResult(nil), a.StepResults...)
	return &cp
}

func assessmentToRecord(a *Assessment) (ports.AssessmentRecord, error) {
	steps, err := json.Marshal(a.Steps)
	if err != nil {
		return ports.AssessmentRecord{}, err
	}
	results, err := json.Marshal(a.StepResults)
	if err != nil {
		return ports.AssessmentRecord{}, err
	}
	return ports.AssessmentRecord{
		ID: string(a.ID), ProjectID: string(a.ProjectID), ScopeID: string(a.ScopeID),
		PrincipalID: string(a.PrincipalID), Title: a.Title, Target: a.Target,
		State: string(a.State), StepsJSON: steps, ResultsJSON: results,
		CreatedAt: a.CreatedAt.UTC(), UpdatedAt: a.UpdatedAt.UTC(),
	}, nil
}

func assessmentFromRecord(rec ports.AssessmentRecord) (*Assessment, error) {
	a := &Assessment{
		ID: contracts.AssessmentID(rec.ID), ProjectID: contracts.ProjectID(rec.ProjectID),
		ScopeID: contracts.ScopeID(rec.ScopeID), PrincipalID: contracts.PrincipalID(rec.PrincipalID),
		Title: rec.Title, Target: rec.Target, State: contracts.AssessmentState(rec.State),
		CreatedAt: rec.CreatedAt.UTC(), UpdatedAt: rec.UpdatedAt.UTC(),
		Cancelled: contracts.AssessmentState(rec.State) == contracts.AssessmentStateCancelled,
	}
	if len(rec.StepsJSON) > 0 {
		if err := json.Unmarshal(rec.StepsJSON, &a.Steps); err != nil {
			return nil, fmt.Errorf("decode assessment steps: %w", err)
		}
	}
	if len(rec.ResultsJSON) > 0 {
		if err := json.Unmarshal(rec.ResultsJSON, &a.StepResults); err != nil {
			return nil, fmt.Errorf("decode assessment results: %w", err)
		}
	}
	return a, nil
}

func (o *Orchestrator) lockRun(id contracts.AssessmentID) func() {
	o.runMu.Lock()
	lock := o.runLocks[id]
	if lock == nil {
		lock = &sync.Mutex{}
		o.runLocks[id] = lock
	}
	o.runMu.Unlock()
	lock.Lock()
	return lock.Unlock
}

// Run executes planned steps sequentially. Each step obtains its own grant.
// Concurrent requests for one assessment are serialized; an interrupted RUNNING
// assessment remains blocked after restart to prevent automatic replay.
func (o *Orchestrator) Run(ctx context.Context, id contracts.AssessmentID) (*Assessment, error) {
	if o == nil || o.issuer == nil || o.broker == nil {
		return nil, fmt.Errorf("%w: orchestrator not configured", contracts.ErrInvalidArgument)
	}
	unlock := o.lockRun(id)
	defer unlock()
	var a *Assessment
	if o.repo != nil {
		// The database transition is atomic across all API replicas. A stale
		// process-local snapshot can never authorize a second execution.
		rec, err := o.repo.ClaimRun(ctx, string(id))
		if err != nil {
			return nil, err
		}
		a, err = assessmentFromRecord(rec)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		a, err = o.Get(id)
		if err != nil {
			return nil, err
		}
		if a.State != contracts.AssessmentStatePlanned {
			return nil, fmt.Errorf("%w: assessment must be PLANNED to run", contracts.ErrInvalidStateTransition)
		}
		if err := a.TransitionTo(contracts.AssessmentStateRunning); err != nil {
			return nil, err
		}
		if err := o.persist(a); err != nil {
			return nil, fmt.Errorf("persist assessment running state: %w", err)
		}
	}
	o.emit(a, "assessment.started", nil)

	failed := false
	for _, step := range a.Steps {
		select {
		case <-ctx.Done():
			if err := a.Cancel(); err != nil {
				return a, fmt.Errorf("cancel assessment after context cancellation: %w", err)
			}
			if err := o.persist(a); err != nil {
				return a, fmt.Errorf("persist cancelled assessment: %w (context: %v)", err, ctx.Err())
			}
			o.emit(a, "assessment.cancelled", map[string]any{"reason": "context done"})
			return a, ctx.Err()
		default:
		}
		if a.Cancelled {
			break
		}
		sr := o.runStep(ctx, a, step)
		a.StepResults = append(a.StepResults, sr)
		// Persist after each completed step so a restart cannot silently lose a
		// completed operation and later repeat it as if it had never happened.
		if err := o.persist(a); err != nil {
			return a, fmt.Errorf("persist assessment step %s result: %w", step.ID, err)
		}
		if !sr.Success {
			failed = true
			// Safe failure: stop sequential chain; assessment marked failed.
			break
		}
	}

	if a.Cancelled {
		// already cancelled
	} else if failed {
		if err := a.TransitionTo(contracts.AssessmentStateFailed); err != nil {
			return a, fmt.Errorf("transition assessment to FAILED: %w", err)
		}
		o.emit(a, "assessment.failed", nil)
	} else {
		if err := a.TransitionTo(contracts.AssessmentStateSucceeded); err != nil {
			return a, fmt.Errorf("transition assessment to SUCCEEDED: %w", err)
		}
		o.emit(a, "assessment.succeeded", nil)
	}
	if err := o.persist(a); err != nil {
		return a, fmt.Errorf("persist terminal assessment state: %w", err)
	}
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
		if eerr != nil {
			sr.Success = false
			sr.ErrorMessage = fmt.Sprintf("persist assessment evidence: %v", eerr)
			sr.Summary = "evidence persistence failed"
			return sr
		}
		sr.EvidenceID = ev.ID
		if o.prov != nil {
			if _, err := o.prov.RecordForEvidence(ev, res.ExecutorID); err != nil {
				sr.Success = false
				sr.ErrorMessage = fmt.Sprintf("persist assessment evidence provenance: %v", err)
				sr.Summary = "evidence provenance persistence failed"
				return sr
			}
		}
		if o.findings != nil {
			_, err := o.findings.Create(findings.CreateRequest{
				ID:            contracts.FindingID(fmt.Sprintf("f-%s-%s", a.ID, step.ID)),
				TaskID:        grant.TaskID,
				ProjectID:     a.ProjectID,
				ScopeID:       a.ScopeID,
				Severity:      contracts.FindingSeverityInfo,
				Title:         "Assessment step: " + step.ID,
				EvidenceIDs:   []contracts.EvidenceID{ev.ID},
				CorrelationID: grant.CorrelationID,
			})
			if err != nil {
				sr.Success = false
				sr.ErrorMessage = fmt.Sprintf("persist assessment finding: %v", err)
				sr.Summary = "finding persistence failed"
				return sr
			}
		}
	}
	if o.phase6 != nil && sr.Success {
		if err := o.phase6.AfterSuccessfulStep(ctx, a, step, grant, res); err != nil {
			sr.Success = false
			sr.ErrorMessage = fmt.Sprintf("assessment result pipeline failed: %v", err)
			sr.Summary = "assessment result persistence failed"
			return sr
		}
	}
	o.emit(a, "assessment.step.completed", map[string]any{
		"step_id": step.ID, "success": sr.Success, "grant_id": string(grant.ID),
	})
	return sr
}

func (o *Orchestrator) persist(a *Assessment) error {
	return o.Save(a)
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

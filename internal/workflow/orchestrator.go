package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"encoding/json"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/lifecycle"
	"cyberos/internal/storage/ports"
	"cyberos/internal/taskengine"
)

// StepAuthorizer obtains a fresh grant for one step. Must call CP path; never cache grants.
type StepAuthorizer interface {
	AuthorizeStep(ctx context.Context, req AuthorizeStepRequest) (contracts.ExecutionGrant, error)
}

// AuthorizeStepRequest is the input for per-step authorization.
type AuthorizeStepRequest struct {
	PrincipalID   contracts.PrincipalID
	ProjectID     contracts.ProjectID
	ScopeID       contracts.ScopeID
	TaskID        contracts.TaskID
	CapabilityID  contracts.CapabilityID
	CorrelationID contracts.CorrelationID
}

// FuncAuthorizer adapts a function (e.g. SDK IssueGrant) as StepAuthorizer.
type FuncAuthorizer struct {
	Fn func(ctx context.Context, req AuthorizeStepRequest) (contracts.ExecutionGrant, error)
}

// AuthorizeStep delegates to Fn.
func (a FuncAuthorizer) AuthorizeStep(ctx context.Context, req AuthorizeStepRequest) (contracts.ExecutionGrant, error) {
	if a.Fn == nil {
		return contracts.ExecutionGrant{}, fmt.Errorf("%w: step authorizer not configured", contracts.ErrInvalidArgument)
	}
	return a.Fn(ctx, req)
}

// Orchestrator runs capability-driven workflows via task engine + lifecycle.
// It never executes tools itself and never reuses grants across steps.
type Orchestrator struct {
	Materializer *Materializer
	Engine       *taskengine.Engine
	Lifecycle    *lifecycle.Controller
	Authorizer   StepAuthorizer

	mu   sync.RWMutex
	runs map[string]*Run
	repo ports.WorkflowRunRepository // optional durable

	Now func() time.Time
}

// WithRepository attaches durable workflow-run storage. Persist errors fail closed.
func (o *Orchestrator) WithRepository(repo ports.WorkflowRunRepository) *Orchestrator {
	if o != nil {
		o.repo = repo
	}
	return o
}

// NewOrchestrator constructs a workflow orchestrator.
func NewOrchestrator(mat *Materializer, eng *taskengine.Engine, life *lifecycle.Controller, auth StepAuthorizer) *Orchestrator {
	return &Orchestrator{
		Materializer: mat,
		Engine:       eng,
		Lifecycle:    life,
		Authorizer:   auth,
		runs:         make(map[string]*Run),
		Now:          func() time.Time { return time.Now().UTC() },
	}
}

// StartRequest begins a workflow run.
type StartRequest struct {
	Definition    Definition
	PrincipalID   contracts.PrincipalID
	RunID         string
	CorrelationID contracts.CorrelationID
}

// Start materializes tasks and registers a PENDING/RUNNING run. Does not execute tools.
func (o *Orchestrator) Start(ctx context.Context, req StartRequest) (*Run, error) {
	if o == nil || o.Materializer == nil {
		return nil, fmt.Errorf("%w: orchestrator not configured", contracts.ErrInvalidArgument)
	}
	mat, err := o.Materializer.Materialize(ctx, req.Definition, req.PrincipalID)
	if err != nil {
		return nil, err
	}
	runID := req.RunID
	if runID == "" {
		runID = fmt.Sprintf("run-%s-%d", req.Definition.ID, o.Now().UnixNano())
	}
	corr := req.CorrelationID
	if corr == "" {
		corr = contracts.CorrelationID(runID)
	}
	run := &Run{
		ID: runID, WorkflowID: req.Definition.ID, Version: req.Definition.Version,
		ProjectID: req.Definition.ProjectID, ScopeID: req.Definition.ScopeID,
		PrincipalID: req.PrincipalID, State: RunStateRunning,
		StepToTask: mat.StepToTask, Steps: map[string]*StepStatus{},
		CorrelationID: corr, StartedAt: o.Now(),
	}
	for _, step := range req.Definition.Steps {
		st := &StepStatus{
			StepID: step.ID, TaskID: mat.StepToTask[step.ID], CapabilityID: step.CapabilityID,
			State: StepPending, ToolID: mat.Resolutions[step.ID].ToolID,
		}
		if len(step.DependsOn) == 0 {
			st.State = StepReady
		}
		run.Steps[step.ID] = st
	}
	if err := run.Validate(); err != nil {
		return nil, err
	}
	o.mu.Lock()
	o.runs[run.ID] = run
	o.mu.Unlock()
	if err := o.persist(run); err != nil {
		return nil, err
	}
	return cloneRun(run), nil
}

// Advance executes the next ready steps (those whose dependencies succeeded).
// Each step gets a NEW CP→Grant decision; no grant caching.
func (o *Orchestrator) Advance(ctx context.Context, runID string, def Definition) (*Run, error) {
	if o == nil {
		return nil, fmt.Errorf("%w: orchestrator not configured", contracts.ErrInvalidArgument)
	}
	o.mu.Lock()
	run, ok := o.runs[runID]
	if !ok {
		o.mu.Unlock()
		if _, err := o.Get(runID); err != nil {
			return nil, err
		}
		o.mu.Lock()
		run = o.runs[runID]
	}
	if run.State == RunStateCancelled || run.State == RunStateSucceeded || run.State == RunStateFailed {
		cp := cloneRun(run)
		o.mu.Unlock()
		return cp, nil
	}
	// Work on a copy of step order from definition
	steps := def.Steps
	o.mu.Unlock()

	for _, step := range steps {
		o.mu.Lock()
		st := run.Steps[step.ID]
		if st == nil || (st.State != StepReady && st.State != StepPending) {
			o.mu.Unlock()
			continue
		}
		// Check dependencies
		ready := true
		for _, dep := range step.DependsOn {
			ds := run.Steps[dep]
			if ds == nil || ds.State != StepSucceeded {
				ready = false
				if ds != nil && (ds.State == StepFailed || ds.State == StepSkipped || ds.State == StepCancelled) {
					st.State = StepSkipped
					st.Error = "dependency not satisfied: " + dep
				}
				break
			}
		}
		if st.State == StepSkipped {
			o.mu.Unlock()
			continue
		}
		if !ready {
			o.mu.Unlock()
			continue
		}
		st.State = StepRunning
		taskID := st.TaskID
		capID := st.CapabilityID
		principal := run.PrincipalID
		project := run.ProjectID
		scope := run.ScopeID
		corr := run.CorrelationID
		params := step.Parameters
		o.mu.Unlock()

		if err := o.executeStep(ctx, run, step.ID, taskID, capID, principal, project, scope, corr, params); err != nil {
			o.mu.Lock()
			run.State = RunStateFailed
			run.Error = err.Error()
			run.FinishedAt = o.Now()
			cp := cloneRun(run)
			o.mu.Unlock()
			return cp, err
		}
	}

	o.mu.Lock()
	o.recomputeRunState(run, def)
	cp := cloneRun(run)
	o.mu.Unlock()
	if err := o.persist(run); err != nil {
		return cp, err
	}
	return cp, nil
}

func (o *Orchestrator) executeStep(
	ctx context.Context,
	run *Run,
	stepID string,
	taskID contracts.TaskID,
	capID contracts.CapabilityID,
	principal contracts.PrincipalID,
	project contracts.ProjectID,
	scope contracts.ScopeID,
	corr contracts.CorrelationID,
	params map[string]any,
) error {
	if o.Authorizer == nil {
		return fmt.Errorf("%w: step authorizer required", contracts.ErrInvalidArgument)
	}
	if o.Lifecycle == nil {
		return fmt.Errorf("%w: lifecycle controller required", contracts.ErrInvalidArgument)
	}

	// Fresh authorization every attempt — never reuse prior step grants.
	grant, err := o.Authorizer.AuthorizeStep(ctx, AuthorizeStepRequest{
		PrincipalID: principal, ProjectID: project, ScopeID: scope,
		TaskID: taskID, CapabilityID: capID, CorrelationID: corr,
	})
	if err != nil {
		o.mu.Lock()
		if st := run.Steps[stepID]; st != nil {
			st.State = StepFailed
			st.Error = err.Error()
		}
		o.mu.Unlock()
		return err
	}

	idem := fmt.Sprintf("%s:%s", run.ID, stepID)
	out, err := o.Lifecycle.Run(lifecycle.RunRequest{
		Grant: grant, Parameters: params, CorrelationID: corr, IdempotencyKey: idem,
	})
	o.mu.Lock()
	defer o.mu.Unlock()
	st := run.Steps[stepID]
	if st == nil {
		return nil
	}
	st.ExecutionID = out.Record.ID
	if out.Evidence != nil {
		st.EvidenceID = *out.Evidence
	}
	if err != nil || out.Record.State != contracts.ExecutionLifecycleSucceeded {
		st.State = StepFailed
		st.Error = firstErr(err, out.Record.FailureReason)
		return fmt.Errorf("step %s failed: %s", stepID, st.Error)
	}
	st.State = StepSucceeded
	return nil
}

func (o *Orchestrator) recomputeRunState(run *Run, def Definition) {
	allDone := true
	anyFailed := false
	anySkipped := false
	for _, step := range def.Steps {
		st := run.Steps[step.ID]
		if st == nil {
			allDone = false
			continue
		}
		switch st.State {
		case StepSucceeded:
		case StepFailed:
			anyFailed = true
		case StepSkipped, StepCancelled:
			anySkipped = true
		default:
			// Mark pending with all deps succeeded as ready
			if st.State == StepPending {
				ok := true
				for _, d := range step.DependsOn {
					ds := run.Steps[d]
					if ds == nil || ds.State != StepSucceeded {
						ok = false
						if ds != nil && (ds.State == StepFailed || ds.State == StepSkipped) {
							st.State = StepSkipped
							st.Error = "dependency not satisfied"
						}
						break
					}
				}
				if ok && st.State == StepPending {
					st.State = StepReady
				}
			}
			if st.State != StepSkipped {
				allDone = false
			}
		}
	}
	if anyFailed {
		run.State = RunStateFailed
		run.FinishedAt = o.Now()
		return
	}
	if allDone {
		if anySkipped {
			run.State = RunStateFailed
		} else {
			run.State = RunStateSucceeded
		}
		run.FinishedAt = o.Now()
	}
}

// Get returns a copy of a run (memory, then durable).
func (o *Orchestrator) Get(runID string) (*Run, error) {
	o.mu.RLock()
	r, ok := o.runs[runID]
	if ok {
		cp := cloneRun(r)
		o.mu.RUnlock()
		return cp, nil
	}
	o.mu.RUnlock()
	if o.repo != nil {
		rec, err := o.repo.GetByID(context.Background(), runID)
		if err != nil {
			return nil, err
		}
		run, err := recordToRun(rec)
		if err != nil {
			return nil, err
		}
		o.mu.Lock()
		o.runs[run.ID] = run
		o.mu.Unlock()
		return cloneRun(run), nil
	}
	return nil, contracts.ErrNotFound
}

// ListByProject returns runs for a project (isolation boundary).
func (o *Orchestrator) ListByProject(projectID contracts.ProjectID, limit, offset int) ([]*Run, error) {
	if o.repo != nil {
		recs, err := o.repo.ListByProject(context.Background(), projectID, limit, offset)
		if err != nil {
			return nil, err
		}
		var out []*Run
		for _, rec := range recs {
			run, err := recordToRun(rec)
			if err != nil {
				return nil, err
			}
			out = append(out, run)
		}
		return out, nil
	}
	o.mu.RLock()
	defer o.mu.RUnlock()
	var out []*Run
	for _, r := range o.runs {
		if r.ProjectID == projectID {
			out = append(out, cloneRun(r))
		}
	}
	return out, nil
}

// Cancel marks the run and remaining steps cancelled (orchestration only).
func (o *Orchestrator) Cancel(runID string) (*Run, error) {
	o.mu.Lock()
	r, ok := o.runs[runID]
	if !ok {
		o.mu.Unlock()
		// try durable load
		if _, err := o.Get(runID); err != nil {
			return nil, err
		}
		o.mu.Lock()
		r = o.runs[runID]
	}
	if r.State == RunStateSucceeded || r.State == RunStateFailed {
		cp := cloneRun(r)
		o.mu.Unlock()
		return cp, nil
	}
	r.State = RunStateCancelled
	r.FinishedAt = o.Now()
	for _, st := range r.Steps {
		if st.State == StepPending || st.State == StepReady || st.State == StepRunning {
			st.State = StepCancelled
		}
	}
	cp := cloneRun(r)
	o.mu.Unlock()
	if err := o.persist(r); err != nil {
		return cp, err
	}
	return cp, nil
}

func (o *Orchestrator) persist(run *Run) error {
	if o == nil || o.repo == nil || run == nil {
		return nil
	}
	rec, err := runToRecord(run)
	if err != nil {
		return err
	}
	return o.repo.Upsert(context.Background(), rec)
}

func runToRecord(run *Run) (ports.WorkflowRunRecord, error) {
	steps, err := json.Marshal(run.Steps)
	if err != nil {
		return ports.WorkflowRunRecord{}, err
	}
	stt, err := json.Marshal(run.StepToTask)
	if err != nil {
		return ports.WorkflowRunRecord{}, err
	}
	return ports.WorkflowRunRecord{
		ID: run.ID, WorkflowID: run.WorkflowID, Version: string(run.Version),
		ProjectID: run.ProjectID, ScopeID: run.ScopeID, PrincipalID: run.PrincipalID,
		State: string(run.State), CorrelationID: string(run.CorrelationID),
		Error: run.Error, StartedAt: run.StartedAt, FinishedAt: run.FinishedAt,
		StepsJSON: steps, StepToTaskJSON: stt,
	}, nil
}

func recordToRun(rec ports.WorkflowRunRecord) (*Run, error) {
	run := &Run{
		ID: rec.ID, WorkflowID: rec.WorkflowID, Version: contracts.SchemaVersion(rec.Version),
		ProjectID: rec.ProjectID, ScopeID: rec.ScopeID, PrincipalID: rec.PrincipalID,
		State: RunState(rec.State), CorrelationID: contracts.CorrelationID(rec.CorrelationID),
		Error: rec.Error, StartedAt: rec.StartedAt, FinishedAt: rec.FinishedAt,
		Steps: map[string]*StepStatus{}, StepToTask: map[string]contracts.TaskID{},
	}
	if len(rec.StepsJSON) > 0 {
		if err := json.Unmarshal(rec.StepsJSON, &run.Steps); err != nil {
			return nil, err
		}
	}
	if len(rec.StepToTaskJSON) > 0 {
		if err := json.Unmarshal(rec.StepToTaskJSON, &run.StepToTask); err != nil {
			return nil, err
		}
	}
	return run, nil
}

func cloneRun(r *Run) *Run {
	cp := *r
	cp.StepToTask = map[string]contracts.TaskID{}
	for k, v := range r.StepToTask {
		cp.StepToTask[k] = v
	}
	cp.Steps = map[string]*StepStatus{}
	for k, v := range r.Steps {
		s := *v
		cp.Steps[k] = &s
	}
	return &cp
}

func firstErr(err error, msg string) string {
	if err != nil {
		return err.Error()
	}
	return msg
}

// Package lifecycle implements the execution lifecycle controller.
// It sits above the Tool Broker, records attributable execution attempts,
// and enriches evidence with safe metadata.
package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/audit"
	"cyberos/internal/contracts"
	"cyberos/internal/evidence"
	"cyberos/internal/findings"
	"cyberos/internal/provenance"
	"cyberos/internal/storage/ports"
)

// Controller orchestrates grant-gated execution and evidence enrichment.
type Controller struct {
	broker     contracts.ToolBroker
	evidence   *evidence.Service
	findings   *findings.Service
	provenance *provenance.Service
	audit      *audit.Service
	policy     *contracts.ResourcePolicy

	mu      sync.RWMutex
	records map[contracts.ExecutionID]*contracts.ExecutionRecord
	repo    ports.ExecutionRecordRepository // optional durable (Phase 9D)
	// Now for tests.
	Now func() time.Time
}

// Option configures the controller.
type Option func(*Controller)

// WithEvidence sets the evidence service.
func WithEvidence(s *evidence.Service) Option { return func(c *Controller) { c.evidence = s } }

// WithFindings sets the findings service.
func WithFindings(s *findings.Service) Option { return func(c *Controller) { c.findings = s } }

// WithProvenance sets the provenance service.
func WithProvenance(s *provenance.Service) Option {
	return func(c *Controller) { c.provenance = s }
}

// WithAudit sets the audit service.
func WithAudit(s *audit.Service) Option { return func(c *Controller) { c.audit = s } }

// WithResourcePolicy attaches a policy snapshot to records (enforcement is sandbox's job).
func WithResourcePolicy(p contracts.ResourcePolicy) Option {
	return func(c *Controller) { c.policy = &p }
}

// WithRepository attaches durable execution record storage. Persist errors propagate.
func WithRepository(repo ports.ExecutionRecordRepository) Option {
	return func(c *Controller) { c.repo = repo }
}

// New constructs a lifecycle controller. broker is required.
func New(broker contracts.ToolBroker, opts ...Option) *Controller {
	c := &Controller{
		broker:  broker,
		records: make(map[contracts.ExecutionID]*contracts.ExecutionRecord),
		Now:     func() time.Time { return time.Now().UTC() },
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// RunRequest is the input to a lifecycle run.
type RunRequest struct {
	Grant           contracts.ExecutionGrant
	Parameters      map[string]any
	Attempt         int
	IdempotencyKey  string               // optional; completed jobs with same key are returned without re-exec
	EvidenceID      contracts.EvidenceID // optional; auto-generated if empty
	FindingTitle    string               // optional: create a finding when non-empty and execution succeeds
	FindingSeverity contracts.FindingSeverity
	CorrelationID   contracts.CorrelationID
}

// RunResult is the outcome of a lifecycle run.
type RunResult struct {
	Record   contracts.ExecutionRecord
	Result   contracts.ExecutionResult
	Evidence *contracts.EvidenceID
	Finding  *contracts.FindingID
}

// Run executes under the broker, records lifecycle, and enriches evidence/findings/provenance.
// No grant ⇒ no execution (broker enforces).
func (c *Controller) Run(req RunRequest) (RunResult, error) {
	if c == nil || c.broker == nil {
		return RunResult{}, fmt.Errorf("%w: lifecycle controller not configured", contracts.ErrInvalidArgument)
	}
	if err := req.Grant.Validate(); err != nil {
		return RunResult{}, fmt.Errorf("%w: grant: %v", contracts.ErrGrantInvalid, err)
	}
	if req.IdempotencyKey != "" {
		existing, found, lookupErr := c.lookupIdempotent(req.IdempotencyKey)
		if lookupErr != nil {
			return RunResult{}, fmt.Errorf("lookup execution idempotency key: %w", lookupErr)
		}
		if found {
			if existing.State == contracts.ExecutionLifecycleStarted || existing.State == contracts.ExecutionLifecycleQueued {
				return RunResult{Record: existing}, fmt.Errorf("%w: execution idempotency key is already in progress or has an uncertain outcome; reconcile before retry", contracts.ErrConflict)
			}
			out := RunResult{Record: existing}
			evidenceID, hasEvidence := existing.Metrics["evidence_id"].(string)
			auditComplete, audited := existing.Metrics["audit_complete"].(bool)
			if !hasEvidence || evidenceID == "" || !audited || !auditComplete || c.evidence == nil {
				return out, fmt.Errorf("%w: prior execution has incomplete durable evidence/audit linkage; reconcile before retry", contracts.ErrConflict)
			}
			evID := contracts.EvidenceID(evidenceID)
			ev, err := c.evidence.Get(evID)
			if err != nil {
				return out, fmt.Errorf("load prior execution evidence %s: %w", evidenceID, err)
			}
			out.Evidence = &evID
			if findingID, ok := existing.Metrics["finding_id"].(string); ok && findingID != "" {
				fid := contracts.FindingID(findingID)
				out.Finding = &fid
			}
			out.Result = contracts.ExecutionResult{
				Status:    contracts.ExecutionStatus(existing.Outcome),
				StartedAt: existing.StartedAt, FinishedAt: existing.FinishedAt,
				ExecutorID: existing.ExecutorID,
				Output:     ev.Payload, ErrorMessage: existing.FailureReason,
				CorrelationID: existing.CorrelationID,
			}
			return out, nil
		}
	}

	now := c.Now()
	attempt := req.Attempt
	if attempt < 1 {
		attempt = 1
	}
	execID := contracts.ExecutionID(fmt.Sprintf("exec-%s-%d-%d", req.Grant.TaskID, attempt, now.UnixNano()))
	corr := req.CorrelationID
	if corr == "" {
		corr = req.Grant.CorrelationID
	}

	rec := contracts.ExecutionRecord{
		ID:             execID,
		State:          contracts.ExecutionLifecycleStarted,
		TaskID:         req.Grant.TaskID,
		GrantID:        req.Grant.ID,
		Principal:      req.Grant.Principal,
		ProjectID:      req.Grant.ProjectID,
		ScopeID:        req.Grant.ScopeID,
		CapabilityID:   req.Grant.CapabilityID,
		Attempt:        attempt,
		StartedAt:      now,
		CorrelationID:  corr,
		IdempotencyKey: req.IdempotencyKey,
		Metrics:        map[string]any{},
	}
	if c.policy != nil {
		p := *c.policy
		rec.ResourcePolicySnapshot = &p
	}
	if err := c.store(rec); err != nil {
		return RunResult{}, err
	}

	// Execute via sole gateway
	execReq := contracts.ExecutionRequest{
		Grant:         req.Grant,
		Parameters:    req.Parameters,
		CorrelationID: corr,
	}
	res, err := c.broker.Execute(execReq)
	finished := c.Now()
	rec.FinishedAt = finished
	rec.Duration = finished.Sub(now)
	rec.ExecutorID = res.ExecutorID
	rec.Outcome = res.Status
	rec.Metrics["duration_ms"] = rec.Duration.Milliseconds()
	rec.Metrics["attempt"] = attempt
	if res.Output != nil {
		rec.Metrics["output_keys"] = len(res.Output)
	}

	switch {
	case err != nil && res.Status == contracts.ExecutionStatusTimeout:
		rec.State = contracts.ExecutionLifecycleTimeout
		rec.FailureReason = res.ErrorMessage
	case err != nil && res.Status == contracts.ExecutionStatusRejected:
		rec.State = contracts.ExecutionLifecycleRejected
		rec.FailureReason = res.ErrorMessage
	case err != nil || res.Status == contracts.ExecutionStatusFailed:
		rec.State = contracts.ExecutionLifecycleFailed
		rec.FailureReason = firstNonEmpty(res.ErrorMessage, errString(err))
	case res.Status == contracts.ExecutionStatusSucceeded:
		rec.State = contracts.ExecutionLifecycleSucceeded
	default:
		rec.State = contracts.ExecutionLifecycleFailed
		rec.FailureReason = res.ErrorMessage
	}
	if err := c.store(rec); err != nil {
		return RunResult{Record: rec, Result: res}, err
	}

	if c.audit != nil {
		if _, auditErr := c.audit.RecordExecution(req.Grant, res); auditErr != nil {
			return RunResult{Record: rec, Result: res}, fmt.Errorf("execution completed but durable audit persistence failed: %w", auditErr)
		}
	}

	out := RunResult{Record: rec, Result: res}

	// Evidence is required for complete forensic linkage (Phase 9D).
	if c.evidence == nil {
		return out, fmt.Errorf("%w: evidence service required for lifecycle runs", contracts.ErrInvalidArgument)
	}
	{
		evID := req.EvidenceID
		if evID == "" {
			evID = contracts.EvidenceID("ev-" + string(execID))
		}
		// Enrich result output with safe lifecycle metadata. Evidence is the
		// canonical replay source, so include stable evidence linkage in payload.
		if res.Output == nil {
			res.Output = map[string]any{}
		}
		res.Output["evidence_id"] = string(evID)
		res.Output["lifecycle"] = map[string]any{
			"execution_id": string(execID),
			"attempt":      attempt,
			"duration_ms":  rec.Duration.Milliseconds(),
			"state":        string(rec.State),
			"executor_id":  res.ExecutorID,
		}
		ev, eerr := c.evidence.RecordFromExecution(res, req.Grant, evID)
		if eerr != nil {
			return out, fmt.Errorf("evidence required: %w", eerr)
		}
		out.Evidence = &ev.ID
		rec.Metrics["evidence_id"] = string(ev.ID)
		if c.provenance != nil {
			if _, perr := c.provenance.RecordForEvidence(ev, res.ExecutorID); perr != nil {
				return out, fmt.Errorf("provenance required: %w", perr)
			}
		}
		if c.audit != nil {
			if _, auditErr := c.audit.RecordEvidence(ev.ID, req.Grant, corr); auditErr != nil {
				return out, fmt.Errorf("evidence recorded but durable audit persistence failed: %w", auditErr)
			}
		}
	}

	// Optional finding on success
	if c.findings != nil && out.Evidence != nil && req.FindingTitle != "" &&
		rec.State == contracts.ExecutionLifecycleSucceeded {
		sev := req.FindingSeverity
		if sev == "" {
			sev = contracts.FindingSeverityInfo
		}
		f, ferr := c.findings.Create(findings.CreateRequest{
			ID:            contracts.FindingID("finding-" + string(execID)),
			TaskID:        req.Grant.TaskID,
			ProjectID:     req.Grant.ProjectID,
			ScopeID:       req.Grant.ScopeID,
			Severity:      sev,
			Title:         req.FindingTitle,
			EvidenceIDs:   []contracts.EvidenceID{*out.Evidence},
			CorrelationID: corr,
		})
		if ferr == nil {
			out.Finding = &f.ID
			rec.Metrics["finding_id"] = string(f.ID)
			if c.provenance != nil {
				if _, provenanceErr := c.provenance.RecordForFinding(f, req.Grant.Principal, req.Grant.CapabilityID, req.Grant.ID, res.ExecutorID); provenanceErr != nil {
					return out, fmt.Errorf("finding recorded but provenance persistence failed: %w", provenanceErr)
				}
			}
			if c.audit != nil {
				p := req.Grant.Principal
				if _, auditErr := c.audit.RecordFinding(f.ID, f.TaskID, f.ProjectID, f.ScopeID, &p, corr); auditErr != nil {
					return out, fmt.Errorf("finding recorded but durable audit persistence failed: %w", auditErr)
				}
			}
		}
	}

	rec.Metrics["audit_complete"] = true
	if err := c.store(rec); err != nil {
		return out, fmt.Errorf("persist completed evidence/audit linkage: %w", err)
	}
	out.Record = rec
	out.Result = res
	if err != nil {
		return out, err
	}
	return out, nil
}

// Get returns a stored execution record.
func (c *Controller) Get(id contracts.ExecutionID) (contracts.ExecutionRecord, error) {
	if c == nil {
		return contracts.ExecutionRecord{}, contracts.ErrInvalidArgument
	}
	if c.repo != nil {
		return c.repo.GetByID(context.Background(), id)
	}
	c.mu.RLock()
	r, ok := c.records[id]
	if ok {
		cp := *r
		c.mu.RUnlock()
		return cp, nil
	}
	c.mu.RUnlock()
	if c.repo != nil {
		return c.repo.GetByID(context.Background(), id)
	}
	return contracts.ExecutionRecord{}, contracts.ErrNotFound
}

func (c *Controller) lookupIdempotent(key string) (contracts.ExecutionRecord, bool, error) {
	if c.repo != nil {
		rec, err := c.repo.GetByIdempotencyKey(context.Background(), key)
		if errors.Is(err, contracts.ErrNotFound) {
			return contracts.ExecutionRecord{}, false, nil
		}
		if err != nil {
			return contracts.ExecutionRecord{}, false, err
		}
		return rec, true, nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, r := range c.records {
		if r.IdempotencyKey == key {
			cp := *r
			return cp, true, nil
		}
	}
	return contracts.ExecutionRecord{}, false, nil
}

// ListByTask returns records for a task (memory, then durable when configured).
func (c *Controller) ListByTask(taskID contracts.TaskID) []contracts.ExecutionRecord {
	if c == nil {
		return nil
	}
	if c.repo != nil {
		out, _ := c.repo.ListByTask(context.Background(), taskID, 200, 0)
		return out
	}
	c.mu.RLock()
	var out []contracts.ExecutionRecord
	seen := map[contracts.ExecutionID]struct{}{}
	for _, r := range c.records {
		if r.TaskID == taskID {
			out = append(out, *r)
			seen[r.ID] = struct{}{}
		}
	}
	c.mu.RUnlock()
	if c.repo != nil {
		if more, err := c.repo.ListByTask(context.Background(), taskID, 200, 0); err == nil {
			for _, r := range more {
				if _, ok := seen[r.ID]; !ok {
					out = append(out, r)
				}
			}
		}
	}
	return out
}

func (c *Controller) store(r contracts.ExecutionRecord) error {
	if c == nil {
		return contracts.ErrInvalidArgument
	}
	if c.repo != nil {
		// The durable repository is authoritative. Persist before publishing to
		// callers and avoid an unbounded/stale process-local duplicate cache.
		return c.repo.Upsert(context.Background(), r)
	}
	c.mu.Lock()
	cp := r
	c.records[r.ID] = &cp
	c.mu.Unlock()
	return nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

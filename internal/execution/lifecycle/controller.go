// Package lifecycle implements the execution lifecycle controller.
// It sits above the Tool Broker, records attributable execution attempts,
// and enriches evidence with safe metadata.
package lifecycle

import (
	"fmt"
	"sync"
	"time"

	"cyberos/internal/audit"
	"cyberos/internal/contracts"
	"cyberos/internal/evidence"
	"cyberos/internal/findings"
	"cyberos/internal/provenance"
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
		ID:            execID,
		State:         contracts.ExecutionLifecycleStarted,
		TaskID:        req.Grant.TaskID,
		GrantID:       req.Grant.ID,
		Principal:     req.Grant.Principal,
		ProjectID:     req.Grant.ProjectID,
		ScopeID:       req.Grant.ScopeID,
		CapabilityID:  req.Grant.CapabilityID,
		Attempt:       attempt,
		StartedAt:     now,
		CorrelationID: corr,
		Metrics:       map[string]any{},
	}
	if c.policy != nil {
		p := *c.policy
		rec.ResourcePolicySnapshot = &p
	}
	c.store(rec)

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
	c.store(rec)

	if c.audit != nil {
		_, _ = c.audit.RecordExecution(req.Grant, res)
	}

	out := RunResult{Record: rec, Result: res}

	// Evidence enrichment
	if c.evidence != nil {
		evID := req.EvidenceID
		if evID == "" {
			evID = contracts.EvidenceID("ev-" + string(execID))
		}
		// Enrich result output with safe lifecycle metadata
		if res.Output == nil {
			res.Output = map[string]any{}
		}
		res.Output["lifecycle"] = map[string]any{
			"execution_id": string(execID),
			"attempt":      attempt,
			"duration_ms":  rec.Duration.Milliseconds(),
			"state":        string(rec.State),
			"executor_id":  res.ExecutorID,
		}
		ev, eerr := c.evidence.RecordFromExecution(res, req.Grant, evID)
		if eerr == nil {
			out.Evidence = &ev.ID
			if c.provenance != nil {
				_, _ = c.provenance.RecordForEvidence(ev, res.ExecutorID)
			}
			if c.audit != nil {
				_, _ = c.audit.RecordEvidence(ev.ID, req.Grant, corr)
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
			if c.provenance != nil {
				_, _ = c.provenance.RecordForFinding(f, req.Grant.Principal, req.Grant.CapabilityID, req.Grant.ID, res.ExecutorID)
			}
			if c.audit != nil {
				p := req.Grant.Principal
				_, _ = c.audit.RecordFinding(f.ID, f.TaskID, f.ProjectID, f.ScopeID, &p, corr)
			}
		}
	}

	if err != nil {
		return out, err
	}
	return out, nil
}

// Get returns a stored execution record.
func (c *Controller) Get(id contracts.ExecutionID) (contracts.ExecutionRecord, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	r, ok := c.records[id]
	if !ok {
		return contracts.ExecutionRecord{}, contracts.ErrNotFound
	}
	return *r, nil
}

// ListByTask returns records for a task.
func (c *Controller) ListByTask(taskID contracts.TaskID) []contracts.ExecutionRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var out []contracts.ExecutionRecord
	for _, r := range c.records {
		if r.TaskID == taskID {
			out = append(out, *r)
		}
	}
	return out
}

func (c *Controller) store(r contracts.ExecutionRecord) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := r
	c.records[r.ID] = &cp
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

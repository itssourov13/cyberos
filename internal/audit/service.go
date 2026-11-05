// Package audit records security-critical decisions and actions.
// The audit trail is append-only in spirit and suitable for forensic reconstruction.
package audit

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/events"
)

// Entry is a single audit record.
type Entry struct {
	ID            contracts.AuditID
	Action        string
	Timestamp     time.Time
	Principal     *contracts.PrincipalRef
	ProjectID     contracts.ProjectID
	ScopeID       contracts.ScopeID
	TaskID        contracts.TaskID
	DecisionID    contracts.DecisionID
	GrantID       contracts.ExecutionGrantID
	CorrelationID contracts.CorrelationID
	CausationID   contracts.CausationID
	Outcome       string // e.g. ALLOWED, DENIED, SUCCEEDED
	Reason        string
	Metadata      map[string]string
}

// Durable is optional append-only persistence for audit entries.
type Durable interface {
	Append(e Entry) error
	ListByCorrelation(id contracts.CorrelationID) ([]Entry, error)
	All() ([]Entry, error)
}

// PagedDurable is an optional efficient read interface implemented by durable
// stores. It avoids materializing the full audit history for browser requests.
type PagedDurable interface {
	List(projectID contracts.ProjectID, limit, offset int) ([]Entry, error)
}

// Service is an append-only audit log that also emits Events.
type Service struct {
	mu      sync.RWMutex
	entries []Entry
	events  *events.Store
	durable Durable
}

// NewService constructs an audit service.
// events store may be nil (audit-only) but production wiring should supply it.
func NewService(eventStore *events.Store) *Service {
	return &Service{
		entries: make([]Entry, 0, 64),
		events:  eventStore,
	}
}

// WithDurable attaches PostgreSQL (or other) append-only audit storage.
func (s *Service) WithDurable(d Durable) *Service {
	if s != nil {
		s.durable = d
	}
	return s
}

// RecordDecision audits a Control Plane decision and emits an event.
func (s *Service) RecordDecision(d contracts.Decision) (Entry, error) {
	action := "controlplane.decision"
	outcome := string(d.Outcome)
	eventType := events.TypeDecisionDenied
	if d.Allowed() {
		eventType = events.TypeDecisionAllowed
	}
	var principal *contracts.PrincipalRef
	if d.Principal.ID != "" {
		p := d.Principal
		principal = &p
	}
	return s.append(Entry{
		ID:            contracts.AuditID("audit-" + string(d.ID)),
		Action:        action,
		Timestamp:     d.Timestamp,
		Principal:     principal,
		ProjectID:     d.ProjectID,
		ScopeID:       d.ScopeID,
		TaskID:        d.TaskID,
		DecisionID:    d.ID,
		CorrelationID: d.CorrelationID,
		Outcome:       outcome,
		Reason:        d.Reason,
	}, eventType, map[string]any{
		"decision_id":   string(d.ID),
		"outcome":       outcome,
		"reason":        d.Reason,
		"capability_id": string(d.CapabilityID),
	})
}

// RecordGrantIssued audits grant issuance.
func (s *Service) RecordGrantIssued(g contracts.ExecutionGrant) (Entry, error) {
	var principal *contracts.PrincipalRef
	if g.Principal.ID != "" {
		p := g.Principal
		principal = &p
	}
	return s.append(Entry{
		ID:            contracts.AuditID("audit-grant-" + string(g.ID)),
		Action:        "grant.issued",
		Timestamp:     g.IssuedAt,
		Principal:     principal,
		ProjectID:     g.ProjectID,
		ScopeID:       g.ScopeID,
		TaskID:        g.TaskID,
		GrantID:       g.ID,
		CorrelationID: g.CorrelationID,
		Outcome:       "ISSUED",
		Reason:        "execution grant issued",
	}, events.TypeGrantIssued, map[string]any{
		"grant_id":      string(g.ID),
		"capability_id": string(g.CapabilityID),
		"expires_at":    g.ExpiresAt.UTC().Format(time.RFC3339),
		"nonce":         g.Nonce,
	})
}

// RecordExecution audits an execution result under a grant.
func (s *Service) RecordExecution(g contracts.ExecutionGrant, res contracts.ExecutionResult) (Entry, error) {
	eventType := events.TypeExecutionSucceeded
	switch res.Status {
	case contracts.ExecutionStatusFailed:
		eventType = events.TypeExecutionFailed
	case contracts.ExecutionStatusRejected:
		eventType = events.TypeExecutionRejected
	}
	var principal *contracts.PrincipalRef
	if g.Principal.ID != "" {
		p := g.Principal
		principal = &p
	}
	return s.append(Entry{
		ID:            contracts.AuditID(fmt.Sprintf("audit-exec-%s-%d", g.ID, res.FinishedAt.UnixNano())),
		Action:        "execution." + string(res.Status),
		Timestamp:     res.FinishedAt,
		Principal:     principal,
		ProjectID:     g.ProjectID,
		ScopeID:       g.ScopeID,
		TaskID:        g.TaskID,
		GrantID:       g.ID,
		CorrelationID: firstNonEmpty(res.CorrelationID, g.CorrelationID),
		Outcome:       string(res.Status),
		Reason:        res.ErrorMessage,
		Metadata:      map[string]string{"executor_id": res.ExecutorID},
	}, eventType, map[string]any{
		"grant_id":    string(g.ID),
		"status":      string(res.Status),
		"executor_id": res.ExecutorID,
		"error":       res.ErrorMessage,
	})
}

// RecordEvidence audits evidence recording.
func (s *Service) RecordEvidence(
	evidenceID contracts.EvidenceID,
	grant contracts.ExecutionGrant,
	correlationID contracts.CorrelationID,
) (Entry, error) {
	var principal *contracts.PrincipalRef
	if grant.Principal.ID != "" {
		p := grant.Principal
		principal = &p
	}
	return s.append(Entry{
		ID:            contracts.AuditID("audit-ev-" + string(evidenceID)),
		Action:        "evidence.recorded",
		Timestamp:     time.Now().UTC(),
		Principal:     principal,
		ProjectID:     grant.ProjectID,
		ScopeID:       grant.ScopeID,
		TaskID:        grant.TaskID,
		GrantID:       grant.ID,
		CorrelationID: firstNonEmpty(correlationID, grant.CorrelationID),
		Outcome:       "RECORDED",
		Reason:        "evidence recorded",
		Metadata:      map[string]string{"evidence_id": string(evidenceID)},
	}, events.TypeEvidenceRecorded, map[string]any{
		"evidence_id": string(evidenceID),
		"grant_id":    string(grant.ID),
	})
}

// RecordFinding audits finding creation.
func (s *Service) RecordFinding(
	findingID contracts.FindingID,
	taskID contracts.TaskID,
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
	principal *contracts.PrincipalRef,
	correlationID contracts.CorrelationID,
) (Entry, error) {
	return s.append(Entry{
		ID:            contracts.AuditID("audit-finding-" + string(findingID)),
		Action:        "finding.created",
		Timestamp:     time.Now().UTC(),
		Principal:     principal,
		ProjectID:     projectID,
		ScopeID:       scopeID,
		TaskID:        taskID,
		CorrelationID: correlationID,
		Outcome:       "CREATED",
		Reason:        "finding created",
		Metadata:      map[string]string{"finding_id": string(findingID)},
	}, events.TypeFindingCreated, map[string]any{
		"finding_id": string(findingID),
		"task_id":    string(taskID),
	})
}

// RecordTaskStateChange audits a task state transition.
func (s *Service) RecordTaskStateChange(
	taskID contracts.TaskID,
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
	from, to contracts.TaskState,
	correlationID contracts.CorrelationID,
) (Entry, error) {
	return s.append(Entry{
		ID:            contracts.AuditID(fmt.Sprintf("audit-task-%s-%s", taskID, to)),
		Action:        "task.state_changed",
		Timestamp:     time.Now().UTC(),
		ProjectID:     projectID,
		ScopeID:       scopeID,
		TaskID:        taskID,
		CorrelationID: correlationID,
		Outcome:       string(to),
		Reason:        fmt.Sprintf("%s -> %s", from, to),
	}, events.TypeTaskStateChanged, map[string]any{
		"task_id": string(taskID),
		"from":    string(from),
		"to":      string(to),
	})
}

// All returns an in-process snapshot. It is intended for diagnostics and
// tests; production API reads must use List so durable storage is authoritative.
func (s *Service) All() []Entry {
	if s == nil {
		return nil
	}
	if s.durable != nil {
		out, err := s.durable.All()
		if err != nil {
			return nil
		}
		return out
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

// List returns newest-first audit entries filtered to one project. An empty
// project ID is reserved for platform operators and includes system entries.
// Durable backends should implement PagedDurable to avoid full-history reads.
func (s *Service) List(projectID contracts.ProjectID, limit, offset int) ([]Entry, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: audit service not configured", contracts.ErrInvalidArgument)
	}
	if limit < 1 || limit > 500 || offset < 0 {
		return nil, fmt.Errorf("%w: invalid audit pagination", contracts.ErrInvalidArgument)
	}
	if s.durable != nil {
		if paged, ok := s.durable.(PagedDurable); ok {
			return paged.List(projectID, limit, offset)
		}
		all, err := s.durable.All()
		if err != nil {
			return nil, err
		}
		return pageEntries(all, projectID, limit, offset), nil
	}
	s.mu.RLock()
	all := make([]Entry, len(s.entries))
	copy(all, s.entries)
	s.mu.RUnlock()
	return pageEntries(all, projectID, limit, offset), nil
}

func pageEntries(all []Entry, projectID contracts.ProjectID, limit, offset int) []Entry {
	filtered := make([]Entry, 0, len(all))
	for _, e := range all {
		if projectID == "" || e.ProjectID == projectID {
			filtered = append(filtered, e)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})
	if offset >= len(filtered) {
		return []Entry{}
	}
	filtered = filtered[offset:]
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered
}

// RecordAdministrative records a platform or credential administration mutation.
func (s *Service) RecordAdministrative(action string, principal contracts.PrincipalRef, project contracts.ProjectID, reason string) (Entry, error) {
	if action == "" || principal.ID == "" {
		return Entry{}, fmt.Errorf("%w: action and actor required", contracts.ErrInvalidArgument)
	}
	now := time.Now().UTC()
	corr := contracts.CorrelationID(fmt.Sprintf("admin-%d", now.UnixNano()))
	p := principal
	return s.append(Entry{
		ID:     contracts.AuditID(fmt.Sprintf("audit-%s-%d", action, now.UnixNano())),
		Action: action, Timestamp: now, Principal: &p, ProjectID: project,
		CorrelationID: corr, Outcome: "SUCCEEDED", Reason: reason,
	}, contracts.EventType("admin.mutation"), map[string]any{"action": action, "reason": reason})
}

// RecordGrantRevoked writes a durable, attributable grant-revocation record.
func (s *Service) RecordGrantRevoked(grantID contracts.ExecutionGrantID, grantPrincipal contracts.PrincipalID, project contracts.ProjectID, correlation contracts.CorrelationID, actor contracts.PrincipalRef, reason string) (Entry, error) {
	if grantID == "" || actor.ID == "" || reason == "" {
		return Entry{}, fmt.Errorf("%w: grant, actor and reason required", contracts.ErrInvalidArgument)
	}
	if correlation == "" {
		correlation = contracts.CorrelationID("grant-revoke-" + string(grantID))
	}
	p := actor
	return s.append(Entry{
		ID:     contracts.AuditID("audit-grant-revoked-" + string(grantID)),
		Action: "grant.revoked", Timestamp: time.Now().UTC(), Principal: &p,
		ProjectID: project, CorrelationID: correlation, GrantID: grantID,
		Outcome: "REVOKED", Reason: reason,
		Metadata: map[string]string{"grant_principal_id": string(grantPrincipal)},
	}, events.TypeGrantRevoked, map[string]any{
		"grant_id": string(grantID), "grant_principal_id": string(grantPrincipal),
		"reason": reason,
	})
}

// RecordTrustMutation audits operator trust/signer lifecycle changes (Phase 21).
// Observability only — does not authorize execution.
func (s *Service) RecordTrustMutation(action string, principal contracts.PrincipalRef, project contracts.ProjectID, digest, keyID, reason, correlation string, meta map[string]string) (Entry, error) {
	if meta == nil {
		meta = map[string]string{}
	}
	if digest != "" {
		meta["pack_content_digest"] = digest
	}
	if keyID != "" {
		meta["pack_key_id"] = keyID
	}
	eventType := contracts.EventType("trust.mutation")
	switch action {
	case "PACK_TRUSTED":
		eventType = events.TypePackTrusted
	case "PACK_REVOKED":
		eventType = events.TypePackRevoked
	case "SIGNER_REGISTERED":
		eventType = events.TypeSignerRegistered
	case "SIGNER_DISABLED":
		eventType = events.TypeSignerDisabled
	case "SIGNER_REVOKED":
		eventType = events.TypeSignerRevoked
	}
	var pref *contracts.PrincipalRef
	if principal.ID != "" {
		p := principal
		pref = &p
	}
	corr := contracts.CorrelationID(correlation)
	if corr == "" {
		corr = contracts.CorrelationID("trust-" + action)
	}
	payload := map[string]any{"action": action, "reason": reason}
	for k, v := range meta {
		payload[k] = v
	}
	return s.append(Entry{
		ID:            contracts.AuditID("audit-trust-" + string(corr) + "-" + action),
		Action:        action,
		Timestamp:     time.Now().UTC(),
		Principal:     pref,
		ProjectID:     project,
		CorrelationID: corr,
		Outcome:       "SUCCEEDED",
		Reason:        reason,
		Metadata:      meta,
	}, eventType, payload)
}

// ByCorrelation returns audit entries for a correlation ID.
func (s *Service) ByCorrelation(id contracts.CorrelationID) []Entry {
	if s == nil {
		return nil
	}
	if s.durable != nil {
		out, err := s.durable.ListByCorrelation(id)
		if err != nil {
			return nil
		}
		return out
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Entry
	for _, e := range s.entries {
		if e.CorrelationID == id {
			out = append(out, e)
		}
	}
	return out
}

// Len returns the number of audit entries.
func (s *Service) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

func (s *Service) append(e Entry, eventType contracts.EventType, payload map[string]any) (Entry, error) {
	if s == nil {
		return Entry{}, fmt.Errorf("%w: audit service not configured", contracts.ErrInvalidArgument)
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	if e.CorrelationID == "" {
		return Entry{}, fmt.Errorf("%w: correlation_id required for audit entry", contracts.ErrInvalidArgument)
	}
	if e.ID == "" {
		return Entry{}, fmt.Errorf("%w: audit id required", contracts.ErrInvalidArgument)
	}

	// Persist first: a failed durable write must not appear as a successful
	// in-memory audit record. Security-critical callers must propagate this error.
	if s.durable != nil {
		if err := s.durable.Append(e); err != nil {
			return Entry{}, err
		}
	}
	// Durable production services read from PostgreSQL and do not retain an
	// unbounded duplicate audit log in process memory.
	if s.durable == nil {
		s.mu.Lock()
		s.entries = append(s.entries, e)
		s.mu.Unlock()
	}

	if s.events != nil {
		var causation contracts.CausationID
		if e.CausationID != "" {
			causation = e.CausationID
		} else if e.DecisionID != "" {
			causation = contracts.CausationID(e.DecisionID)
		}
		_, _ = s.events.Append(events.AppendRequest{
			ID:            contracts.EventID("evt-" + string(e.ID)),
			Type:          eventType,
			Principal:     e.Principal,
			ProjectID:     e.ProjectID,
			ScopeID:       e.ScopeID,
			TaskID:        e.TaskID,
			CorrelationID: e.CorrelationID,
			CausationID:   causation,
			Payload:       payload,
			Timestamp:     e.Timestamp,
		})
	}
	return e, nil
}

func firstNonEmpty(a, b contracts.CorrelationID) contracts.CorrelationID {
	if a != "" {
		return a
	}
	return b
}

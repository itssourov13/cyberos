// Package cases provides Security Case management (reporting layer only).
// Cases never authorize or execute tools.
package cases

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/riskprio"
	"cyberos/internal/storage/ports"
)

// opTimeout bounds every repository round trip made by the service.
const opTimeout = 10 * time.Second

// Case is a management container for findings/assets under an assessment.
type Case struct {
	ID           string
	TenantID     contracts.TenantID
	ProjectID    contracts.ProjectID
	AssessmentID string
	Title        string
	Description  string
	Status       contracts.CaseStatus
	Priority     contracts.RiskPriority
	FindingIDs   []contracts.FindingID
	AssetIDs     []contracts.AssetID
	CreatedBy    contracts.PrincipalID
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ResolvedAt   *time.Time
	Metadata     map[string]string
}

// StatusChange is an attributable status transition.
type StatusChange = ports.CaseStatusChange

// Service manages cases. With a repository attached the repository is the
// single source of truth (no in-memory shadow state can override it); without
// one (development/tests) an in-memory store is used.
type Service struct {
	mu      sync.RWMutex
	items   map[string]*Case
	history map[string][]StatusChange
	repo    ports.CaseRepository
}

// NewService constructs a case service.
func NewService() *Service {
	return &Service{items: map[string]*Case{}, history: map[string][]StatusChange{}}
}

// WithRepository attaches durable persistence. Once attached, all reads and
// writes go through the repository.
func (s *Service) WithRepository(repo ports.CaseRepository) *Service {
	if s != nil {
		s.repo = repo
	}
	return s
}

func ctxTO() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), opTimeout)
}

func toRecord(c *Case) ports.CaseRecord {
	return ports.CaseRecord{
		ID: c.ID, TenantID: c.TenantID, ProjectID: c.ProjectID, AssessmentID: c.AssessmentID,
		Title: c.Title, Description: c.Description, Status: c.Status, Priority: c.Priority,
		FindingIDs: c.FindingIDs, AssetIDs: c.AssetIDs, CreatedBy: c.CreatedBy,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, ResolvedAt: c.ResolvedAt, Metadata: c.Metadata,
	}
}

func recordToCase(r ports.CaseRecord) *Case {
	return &Case{
		ID: r.ID, TenantID: r.TenantID, ProjectID: r.ProjectID, AssessmentID: r.AssessmentID,
		Title: r.Title, Description: r.Description, Status: r.Status, Priority: r.Priority,
		FindingIDs: r.FindingIDs, AssetIDs: r.AssetIDs, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, ResolvedAt: r.ResolvedAt, Metadata: r.Metadata,
	}
}

func clone(c *Case) *Case {
	cp := *c
	cp.FindingIDs = append([]contracts.FindingID(nil), c.FindingIDs...)
	cp.AssetIDs = append([]contracts.AssetID(nil), c.AssetIDs...)
	if c.Metadata != nil {
		cp.Metadata = make(map[string]string, len(c.Metadata))
		for k, v := range c.Metadata {
			cp.Metadata[k] = v
		}
	}
	return &cp
}

// CreateRequest is input for a new case. TenantID and CreatedBy MUST be set by
// trusted server code from authenticated identity and the canonical project
// record; they are never accepted from a client payload.
type CreateRequest struct {
	ID           string
	TenantID     contracts.TenantID
	ProjectID    contracts.ProjectID
	AssessmentID string
	Title        string
	Description  string
	FindingIDs   []contracts.FindingID
	AssetIDs     []contracts.AssetID
	CreatedBy    contracts.PrincipalID
	Priority     contracts.RiskPriority
}

// Create opens a new case. It never overwrites an existing case id.
func (s *Service) Create(req CreateRequest) (*Case, error) {
	if req.ID == "" || req.ProjectID == "" || req.Title == "" {
		return nil, fmt.Errorf("%w: id, project, title required", contracts.ErrInvalidArgument)
	}
	now := time.Now().UTC()
	prio := req.Priority
	if !prio.Valid() {
		prio = contracts.RiskPriorityMedium
	}
	c := &Case{
		ID: req.ID, TenantID: req.TenantID, ProjectID: req.ProjectID,
		AssessmentID: req.AssessmentID, Title: req.Title, Description: req.Description,
		Status: contracts.CaseStatusOpen, Priority: prio,
		FindingIDs: append([]contracts.FindingID{}, req.FindingIDs...),
		AssetIDs:   append([]contracts.AssetID{}, req.AssetIDs...),
		CreatedBy:  req.CreatedBy, CreatedAt: now, UpdatedAt: now,
		Metadata: map[string]string{},
	}
	if s.repo != nil {
		ctx, cancel := ctxTO()
		defer cancel()
		if err := s.repo.Insert(ctx, toRecord(c)); err != nil {
			return nil, err
		}
		return clone(c), nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[c.ID]; ok {
		return nil, contracts.ErrConflict
	}
	s.items[c.ID] = c
	return clone(c), nil
}

// Get returns a case by id.
func (s *Service) Get(id string) (*Case, error) {
	if s.repo != nil {
		ctx, cancel := ctxTO()
		defer cancel()
		rec, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return recordToCase(rec), nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	return clone(c), nil
}

// ListByProject returns cases for a project (repository-backed when durable).
func (s *Service) ListByProject(projectID contracts.ProjectID) []*Case {
	if s.repo != nil {
		ctx, cancel := ctxTO()
		defer cancel()
		recs, err := s.repo.ListByProject(ctx, projectID, 500, 0)
		if err != nil {
			return nil
		}
		out := make([]*Case, 0, len(recs))
		for _, r := range recs {
			out = append(out, recordToCase(r))
		}
		return out
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Case
	for _, c := range s.items {
		if c.ProjectID == projectID {
			out = append(out, clone(c))
		}
	}
	return out
}

// ListByAssessment returns cases for an assessment.
func (s *Service) ListByAssessment(assessmentID string) []*Case {
	if s.repo != nil {
		ctx, cancel := ctxTO()
		defer cancel()
		recs, err := s.repo.ListByAssessment(ctx, assessmentID, 500, 0)
		if err != nil {
			return nil
		}
		out := make([]*Case, 0, len(recs))
		for _, r := range recs {
			out = append(out, recordToCase(r))
		}
		return out
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Case
	for _, c := range s.items {
		if c.AssessmentID == assessmentID {
			out = append(out, clone(c))
		}
	}
	return out
}

// TransitionStatus applies a transition attributed to the "system" actor.
// Prefer TransitionStatusBy so history names the real principal.
func (s *Service) TransitionStatus(id string, next contracts.CaseStatus) (*Case, error) {
	return s.TransitionStatusBy(id, next, "system", "")
}

// TransitionStatusBy validates and applies a status transition and records an
// attributable history entry. With a repository the update and the history
// row commit in one transaction.
func (s *Service) TransitionStatusBy(id string, next contracts.CaseStatus, actor contracts.PrincipalID, reason string) (*Case, error) {
	if !next.Valid() {
		return nil, fmt.Errorf("%w: invalid status", contracts.ErrInvalidArgument)
	}
	if actor == "" {
		return nil, fmt.Errorf("%w: actor required", contracts.ErrInvalidArgument)
	}
	now := time.Now().UTC()
	if s.repo != nil {
		ctx, cancel := ctxTO()
		defer cancel()
		rec, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if !allowedTransition(rec.Status, next) {
			return nil, fmt.Errorf("%w: cannot transition %s → %s", contracts.ErrInvalidArgument, rec.Status, next)
		}
		if rec.Status == next {
			return recordToCase(rec), nil
		}
		out, err := s.repo.Transition(ctx, id, rec.Status, next, ports.CaseStatusChange{
			CaseID: id, TenantID: rec.TenantID, ProjectID: rec.ProjectID,
			From: rec.Status, To: next, Actor: actor, Reason: reason, At: now,
		})
		if err != nil {
			return nil, err
		}
		return recordToCase(out), nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	if !allowedTransition(c.Status, next) {
		return nil, fmt.Errorf("%w: cannot transition %s → %s", contracts.ErrInvalidArgument, c.Status, next)
	}
	if c.Status == next {
		return clone(c), nil
	}
	prev := c.Status
	c.Status = next
	c.UpdatedAt = now
	if next == contracts.CaseStatusResolved || next == contracts.CaseStatusAccepted {
		t := now
		c.ResolvedAt = &t
	}
	if next == contracts.CaseStatusReopened {
		c.ResolvedAt = nil
	}
	s.history[id] = append(s.history[id], StatusChange{
		CaseID: id, TenantID: c.TenantID, ProjectID: c.ProjectID,
		From: prev, To: next, Actor: actor, Reason: reason, At: now,
	})
	return clone(c), nil
}

// History returns the attributable status history of a case, oldest first.
func (s *Service) History(id string) ([]StatusChange, error) {
	if s.repo != nil {
		ctx, cancel := ctxTO()
		defer cancel()
		return s.repo.StatusHistory(ctx, id)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]StatusChange(nil), s.history[id]...), nil
}

// mutate loads a case, applies fn and persists the change.
func (s *Service) mutate(id string, fn func(*Case)) (*Case, error) {
	if s.repo != nil {
		ctx, cancel := ctxTO()
		defer cancel()
		rec, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		c := recordToCase(rec)
		fn(c)
		c.UpdatedAt = time.Now().UTC()
		if err := s.repo.Upsert(ctx, toRecord(c)); err != nil {
			return nil, err
		}
		return c, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	fn(c)
	c.UpdatedAt = time.Now().UTC()
	return clone(c), nil
}

// LinkFinding attaches a finding reference.
func (s *Service) LinkFinding(id string, findingID contracts.FindingID) (*Case, error) {
	return s.mutate(id, func(c *Case) {
		for _, f := range c.FindingIDs {
			if f == findingID {
				return
			}
		}
		c.FindingIDs = append(c.FindingIDs, findingID)
	})
}

// LinkAsset attaches an asset reference.
func (s *Service) LinkAsset(id string, assetID contracts.AssetID) (*Case, error) {
	return s.mutate(id, func(c *Case) {
		for _, a := range c.AssetIDs {
			if a == assetID {
				return
			}
		}
		c.AssetIDs = append(c.AssetIDs, assetID)
	})
}

// DerivePriorityFromRisks sets case priority to the max of linked risk evaluations.
func (s *Service) DerivePriorityFromRisks(id string, engine *riskprio.Engine) (*Case, error) {
	return s.mutate(id, func(c *Case) {
		if engine == nil {
			return
		}
		maxScore := -1
		best := c.Priority
		for _, fid := range c.FindingIDs {
			ev, err := engine.GetByFinding(fid)
			if err != nil {
				continue
			}
			if ev.Score > maxScore {
				maxScore = ev.Score
				best = ev.Priority
			}
		}
		c.Priority = best
	})
}

func allowedTransition(from, to contracts.CaseStatus) bool {
	if from == to {
		return true
	}
	allowed := map[contracts.CaseStatus][]contracts.CaseStatus{
		contracts.CaseStatusOpen: {
			contracts.CaseStatusAcknowledged, contracts.CaseStatusInProgress,
			contracts.CaseStatusResolved, contracts.CaseStatusAccepted,
		},
		contracts.CaseStatusAcknowledged: {
			contracts.CaseStatusInProgress, contracts.CaseStatusResolved, contracts.CaseStatusAccepted,
		},
		contracts.CaseStatusInProgress: {
			contracts.CaseStatusResolved, contracts.CaseStatusAccepted,
		},
		contracts.CaseStatusResolved: {contracts.CaseStatusReopened},
		contracts.CaseStatusAccepted: {contracts.CaseStatusReopened},
		contracts.CaseStatusReopened: {
			contracts.CaseStatusInProgress, contracts.CaseStatusResolved, contracts.CaseStatusAccepted,
		},
	}
	for _, n := range allowed[from] {
		if n == to {
			return true
		}
	}
	return false
}

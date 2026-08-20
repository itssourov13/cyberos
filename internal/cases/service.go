// Package cases provides Security Case management (reporting layer only).
// Cases never authorize or execute tools.
package cases

import (
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/riskprio"
)

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

// Service manages cases in-memory (PG migration present for durable wire-up).
type Service struct {
	mu    sync.RWMutex
	items map[string]*Case
}

// NewService constructs a case service.
func NewService() *Service {
	return &Service{items: map[string]*Case{}}
}

// CreateRequest is input for a new case.
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

// Create opens a new case.
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[c.ID]; ok {
		return nil, contracts.ErrConflict
	}
	s.items[c.ID] = c
	cp := *c
	return &cp, nil
}

// Get returns a case by id.
func (s *Service) Get(id string) (*Case, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

// ListByProject returns cases for a project.
func (s *Service) ListByProject(projectID contracts.ProjectID) []*Case {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Case
	for _, c := range s.items {
		if c.ProjectID == projectID {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out
}

// ListByAssessment returns cases for an assessment.
func (s *Service) ListByAssessment(assessmentID string) []*Case {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*Case
	for _, c := range s.items {
		if c.AssessmentID == assessmentID {
			cp := *c
			out = append(out, &cp)
		}
	}
	return out
}

// TransitionStatus validates and applies a status transition.
func (s *Service) TransitionStatus(id string, next contracts.CaseStatus) (*Case, error) {
	if !next.Valid() {
		return nil, fmt.Errorf("%w: invalid status", contracts.ErrInvalidArgument)
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
	c.Status = next
	c.UpdatedAt = time.Now().UTC()
	if next == contracts.CaseStatusResolved || next == contracts.CaseStatusAccepted {
		t := c.UpdatedAt
		c.ResolvedAt = &t
	}
	if next == contracts.CaseStatusReopened {
		c.ResolvedAt = nil
	}
	cp := *c
	return &cp, nil
}

// LinkFinding attaches a finding reference.
func (s *Service) LinkFinding(id string, findingID contracts.FindingID) (*Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	for _, f := range c.FindingIDs {
		if f == findingID {
			cp := *c
			return &cp, nil
		}
	}
	c.FindingIDs = append(c.FindingIDs, findingID)
	c.UpdatedAt = time.Now().UTC()
	cp := *c
	return &cp, nil
}

// LinkAsset attaches an asset reference.
func (s *Service) LinkAsset(id string, assetID contracts.AssetID) (*Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	for _, a := range c.AssetIDs {
		if a == assetID {
			cp := *c
			return &cp, nil
		}
	}
	c.AssetIDs = append(c.AssetIDs, assetID)
	c.UpdatedAt = time.Now().UTC()
	cp := *c
	return &cp, nil
}

// DerivePriorityFromRisks sets case priority to the max of linked risk evaluations.
func (s *Service) DerivePriorityFromRisks(id string, engine *riskprio.Engine) (*Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	if engine == nil {
		return c, nil
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
	c.UpdatedAt = time.Now().UTC()
	cp := *c
	return &cp, nil
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

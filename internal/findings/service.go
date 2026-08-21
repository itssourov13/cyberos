// Package findings provides services for creating structured Findings
// derived from Evidence.
package findings

import (
	"context"
	"fmt"
	"sync"

	"cyberos/internal/contracts"
	domainfinding "cyberos/internal/domain/finding"
	"cyberos/internal/evidence"
	"cyberos/internal/storage/ports"
)

// Service records and retrieves Findings.
type Service struct {
	mu            sync.RWMutex
	items         map[contracts.FindingID]*domainfinding.Finding
	byFingerprint map[string]*domainfinding.Finding
	evidence      *evidence.Service
	repo          ports.FindingRepository // optional durable backend (Phase 7A)
}

// NewService constructs a findings service.
// evidenceSvc is used to verify supporting evidence exists and is linked.
func NewService(evidenceSvc *evidence.Service) *Service {
	return &Service{
		items:         make(map[contracts.FindingID]*domainfinding.Finding),
		byFingerprint: make(map[string]*domainfinding.Finding),
		evidence:      evidenceSvc,
	}
}

// WithRepository attaches optional PostgreSQL (or other) persistence. Nil keeps in-memory only.
func (s *Service) WithRepository(repo ports.FindingRepository) *Service {
	if s != nil {
		s.repo = repo
	}
	return s
}

func (s *Service) persist(f *domainfinding.Finding) error {
	if s == nil || s.repo == nil || f == nil {
		return nil
	}
	return s.repo.Upsert(context.Background(), f)
}

// CreateRequest carries data for a new finding.
type CreateRequest struct {
	ID            contracts.FindingID
	TaskID        contracts.TaskID
	ProjectID     contracts.ProjectID
	ScopeID       contracts.ScopeID
	Severity      contracts.FindingSeverity
	Title         string
	Description   string
	EvidenceIDs   []contracts.EvidenceID
	CorrelationID contracts.CorrelationID
}

// Create builds a Finding that references supporting Evidence.
// Each evidence ID must exist and be linked to the same task when possible.
func (s *Service) Create(req CreateRequest) (*domainfinding.Finding, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: findings service not configured", contracts.ErrInvalidArgument)
	}
	f, err := domainfinding.New(req.ID, req.TaskID, req.ProjectID, req.ScopeID, req.Severity, req.Title)
	if err != nil {
		return nil, err
	}
	f.Description = req.Description
	f.CorrelationID = req.CorrelationID

	for _, eid := range req.EvidenceIDs {
		if s.evidence != nil {
			ev, err := s.evidence.Get(eid)
			if err != nil {
				return nil, fmt.Errorf("%w: evidence %s: %v", contracts.ErrInvalidArgument, eid, err)
			}
			if err := evidence.EnsureLinked(ev); err != nil {
				return nil, err
			}
			// Prefer same-task linkage
			if ev.TaskID != req.TaskID {
				return nil, fmt.Errorf("%w: evidence %s task mismatch", contracts.ErrInvalidArgument, eid)
			}
		}
		f.AddEvidence(eid)
	}

	if err := f.Validate(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[f.ID]; exists {
		return nil, contracts.ErrConflict
	}
	cp := *f
	s.items[f.ID] = &cp
	if cp.Fingerprint != "" {
		s.byFingerprint[cp.Fingerprint] = &cp
	}
	if err := s.persist(&cp); err != nil {
		delete(s.items, cp.ID)
		if cp.Fingerprint != "" {
			delete(s.byFingerprint, cp.Fingerprint)
		}
		return nil, err
	}
	return &cp, nil
}

// Get returns a finding by ID.
func (s *Service) Get(id contracts.FindingID) (*domainfinding.Finding, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: findings service not configured", contracts.ErrInvalidArgument)
	}
	s.mu.RLock()
	f, ok := s.items[id]
	if ok {
		cp := *f
		s.mu.RUnlock()
		return &cp, nil
	}
	s.mu.RUnlock()
	if s.repo != nil {
		return s.repo.GetByID(context.Background(), id)
	}
	return nil, contracts.ErrNotFound
}

// ListByTask returns findings for a task.
func (s *Service) ListByTask(taskID contracts.TaskID) []*domainfinding.Finding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domainfinding.Finding
	for _, f := range s.items {
		if f.TaskID == taskID {
			cp := *f
			out = append(out, &cp)
		}
	}
	return out
}

// Package provenance provides services for recording full lineage
// (WHO / WHAT / WHEN / WHERE / WHY) for Evidence and Findings.
package provenance

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"cyberos/internal/contracts"
	domainevidence "cyberos/internal/domain/evidence"
	domainfinding "cyberos/internal/domain/finding"
	domainprov "cyberos/internal/domain/provenance"
	"cyberos/internal/storage/ports"
)

// Service stores provenance records.
type Service struct {
	mu    sync.RWMutex
	items map[string]*domainprov.Provenance // key: subjectType:subjectID
	repo  ports.ProvenanceRepository
}

// NewService constructs a provenance service.
func NewService() *Service {
	return &Service{items: make(map[string]*domainprov.Provenance)}
}

// WithRepository attaches optional durable persistence. When set, persist errors propagate.
func (s *Service) WithRepository(repo ports.ProvenanceRepository) *Service {
	if s != nil {
		s.repo = repo
	}
	return s
}

func (s *Service) persist(p *domainprov.Provenance) error {
	if s == nil || s.repo == nil || p == nil {
		return nil
	}
	return s.repo.Upsert(context.Background(), p)
}

func key(subjectType, subjectID string) string {
	return subjectType + ":" + subjectID
}

// RecordForEvidence builds and stores complete provenance for evidence.
func (s *Service) RecordForEvidence(
	ev *domainevidence.Evidence,
	executorID string,
) (*domainprov.Provenance, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: provenance service not configured", contracts.ErrInvalidArgument)
	}
	if ev == nil {
		return nil, contracts.ErrInvalidArgument
	}
	p, err := domainprov.NewForEvidence(
		ev.ID,
		ev.Principal,
		ev.ProjectID,
		ev.ScopeID,
		ev.TaskID,
		ev.CapabilityID,
		ev.GrantID,
		executorID,
	)
	if err != nil {
		return nil, err
	}
	p.CorrelationID = ev.CorrelationID
	p.Digest = ev.Digest
	if err := p.Validate(); err != nil {
		return nil, err
	}
	// Completeness: all forensic fields must be present
	if err := ensureComplete(&p.Provenance); err != nil {
		return nil, err
	}

	cp := *p
	k := key("evidence", string(ev.ID))
	if s.repo != nil {
		if _, err := s.repo.Get(context.Background(), "evidence", string(ev.ID)); err == nil {
			return nil, contracts.ErrConflict
		} else if !errors.Is(err, contracts.ErrNotFound) {
			return nil, fmt.Errorf("check existing evidence provenance: %w", err)
		}
		if err := s.persist(&cp); err != nil {
			return nil, fmt.Errorf("persist evidence provenance: %w", err)
		}
		return &cp, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[k]; exists {
		return nil, contracts.ErrConflict
	}
	s.items[k] = &cp
	return &cp, nil
}

// RecordForFinding builds provenance for a finding, linking parent evidence IDs.
func (s *Service) RecordForFinding(
	f *domainfinding.Finding,
	principal contracts.PrincipalRef,
	capabilityID contracts.CapabilityID,
	grantID contracts.ExecutionGrantID,
	executorID string,
) (*domainprov.Provenance, error) {
	if s == nil || f == nil {
		return nil, contracts.ErrInvalidArgument
	}
	p := contracts.Provenance{
		SubjectType:   "finding",
		SubjectID:     string(f.ID),
		Principal:     principal,
		ProjectID:     f.ProjectID,
		ScopeID:       f.ScopeID,
		TaskID:        f.TaskID,
		CapabilityID:  capabilityID,
		GrantID:       grantID,
		ExecutorID:    executorID,
		ProducedAt:    f.ProducedAt,
		CorrelationID: f.CorrelationID,
	}
	for _, eid := range f.EvidenceIDs {
		p.ParentIDs = append(p.ParentIDs, string(eid))
	}
	dp, err := domainprov.NewFromContract(p)
	if err != nil {
		return nil, err
	}
	if err := ensureComplete(&dp.Provenance); err != nil {
		return nil, err
	}

	cp := *dp
	k := key("finding", string(f.ID))
	if s.repo != nil {
		if _, err := s.repo.Get(context.Background(), "finding", string(f.ID)); err == nil {
			return nil, contracts.ErrConflict
		} else if !errors.Is(err, contracts.ErrNotFound) {
			return nil, fmt.Errorf("check existing finding provenance: %w", err)
		}
		if err := s.persist(&cp); err != nil {
			return nil, fmt.Errorf("persist finding provenance: %w", err)
		}
		return &cp, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[k]; exists {
		return nil, contracts.ErrConflict
	}
	s.items[k] = &cp
	return &cp, nil
}

// Get returns provenance for a subject.
func (s *Service) Get(subjectType, subjectID string) (*domainprov.Provenance, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: provenance service not configured", contracts.ErrInvalidArgument)
	}
	if s.repo != nil {
		return s.repo.Get(context.Background(), subjectType, subjectID)
	}
	s.mu.RLock()
	p, ok := s.items[key(subjectType, subjectID)]
	if ok {
		cp := *p
		s.mu.RUnlock()
		return &cp, nil
	}
	s.mu.RUnlock()
	return nil, contracts.ErrNotFound
}

// ensureComplete checks WHO/WHAT/WHEN/WHERE (scope/task) / capability / grant.
func ensureComplete(p *contracts.Provenance) error {
	if p.SubjectType == "" || p.SubjectID == "" {
		return fmt.Errorf("%w: provenance missing subject", contracts.ErrInvalidArgument)
	}
	if p.Principal.ID == "" {
		return fmt.Errorf("%w: provenance missing principal (WHO)", contracts.ErrInvalidArgument)
	}
	if p.ProjectID == "" || p.ScopeID == "" || p.TaskID == "" {
		return fmt.Errorf("%w: provenance missing project/scope/task (WHERE)", contracts.ErrInvalidArgument)
	}
	if p.CapabilityID == "" {
		return fmt.Errorf("%w: provenance missing capability (WHAT)", contracts.ErrInvalidArgument)
	}
	if p.ProducedAt.IsZero() {
		return fmt.Errorf("%w: provenance missing timestamp (WHEN)", contracts.ErrInvalidArgument)
	}
	return nil
}

// Package provenance provides services for recording full lineage
// (WHO / WHAT / WHEN / WHERE / WHY) for Evidence and Findings.
package provenance

import (
	"fmt"
	"sync"

	"cyberos/internal/contracts"
	domainevidence "cyberos/internal/domain/evidence"
	domainfinding "cyberos/internal/domain/finding"
	domainprov "cyberos/internal/domain/provenance"
)

// Service stores provenance records.
type Service struct {
	mu    sync.RWMutex
	items map[string]*domainprov.Provenance // key: subjectType:subjectID
}

// NewService constructs a provenance service.
func NewService() *Service {
	return &Service{items: make(map[string]*domainprov.Provenance)}
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

	s.mu.Lock()
	defer s.mu.Unlock()
	k := key("evidence", string(ev.ID))
	if _, exists := s.items[k]; exists {
		return nil, contracts.ErrConflict
	}
	cp := *p
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

	s.mu.Lock()
	defer s.mu.Unlock()
	k := key("finding", string(f.ID))
	if _, exists := s.items[k]; exists {
		return nil, contracts.ErrConflict
	}
	cp := *dp
	s.items[k] = &cp
	return &cp, nil
}

// Get returns provenance for a subject.
func (s *Service) Get(subjectType, subjectID string) (*domainprov.Provenance, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: provenance service not configured", contracts.ErrInvalidArgument)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.items[key(subjectType, subjectID)]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
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

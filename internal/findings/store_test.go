package findings

import (
	"context"
	"testing"

	"cyberos/internal/contracts"
	domainfinding "cyberos/internal/domain/finding"
)

type memFindingRepo struct {
	byID map[contracts.FindingID]*domainfinding.Finding
	byFP map[string]*domainfinding.Finding
}

func (m *memFindingRepo) Upsert(_ context.Context, f *domainfinding.Finding) error {
	if m.byID == nil {
		m.byID = map[contracts.FindingID]*domainfinding.Finding{}
		m.byFP = map[string]*domainfinding.Finding{}
	}
	cp := *f
	m.byID[f.ID] = &cp
	if f.Fingerprint != "" {
		m.byFP[string(f.ProjectID)+"|"+f.Fingerprint] = &cp
	}
	return nil
}
func (m *memFindingRepo) GetByID(_ context.Context, id contracts.FindingID) (*domainfinding.Finding, error) {
	f, ok := m.byID[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *f
	return &cp, nil
}
func (m *memFindingRepo) GetByFingerprint(_ context.Context, projectID contracts.ProjectID, fp string) (*domainfinding.Finding, error) {
	f, ok := m.byFP[string(projectID)+"|"+fp]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *f
	return &cp, nil
}
func (m *memFindingRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainfinding.Finding, error) {
	return nil, nil
}
func (m *memFindingRepo) ListByAssessment(context.Context, string, int, int) ([]*domainfinding.Finding, error) {
	return nil, nil
}

func TestService_WithRepositoryPersistAndReadThrough(t *testing.T) {
	repo := &memFindingRepo{}
	s := NewService(nil).WithRepository(repo)
	f, err := s.Create(CreateRequest{
		ID: "f1", TaskID: "t1", ProjectID: "p1", ScopeID: "s1",
		Severity: contracts.FindingSeverityMedium, Title: "T",
	})
	if err != nil {
		t.Fatal(err)
	}
	// clear in-memory
	s.items = map[contracts.FindingID]*domainfinding.Finding{}
	s.byFingerprint = map[string]*domainfinding.Finding{}
	got, err := s.Get(f.ID)
	if err != nil || got.Title != "T" {
		t.Fatal(err, got)
	}
}

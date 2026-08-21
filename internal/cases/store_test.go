package cases

import (
	"context"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

type memCaseRepo struct {
	m map[string]ports.CaseRecord
}

func (m *memCaseRepo) Upsert(_ context.Context, c ports.CaseRecord) error {
	if m.m == nil {
		m.m = map[string]ports.CaseRecord{}
	}
	m.m[c.ID] = c
	return nil
}
func (m *memCaseRepo) GetByID(_ context.Context, id string) (ports.CaseRecord, error) {
	c, ok := m.m[id]
	if !ok {
		return ports.CaseRecord{}, contracts.ErrNotFound
	}
	return c, nil
}
func (m *memCaseRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]ports.CaseRecord, error) {
	return nil, nil
}
func (m *memCaseRepo) ListByAssessment(context.Context, string, int, int) ([]ports.CaseRecord, error) {
	return nil, nil
}
func (m *memCaseRepo) ListByTenant(context.Context, contracts.TenantID, int, int) ([]ports.CaseRecord, error) {
	return nil, nil
}

func TestCase_WithRepositoryReadThrough(t *testing.T) {
	repo := &memCaseRepo{}
	s := NewService().WithRepository(repo)
	c, err := s.Create(CreateRequest{ID: "c1", ProjectID: "p1", Title: "T", TenantID: "ten-a"})
	if err != nil {
		t.Fatal(err)
	}
	s.items = map[string]*Case{}
	got, err := s.Get(c.ID)
	if err != nil || got.Title != "T" {
		t.Fatal(err, got)
	}
}

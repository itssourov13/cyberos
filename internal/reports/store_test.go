package reports

import (
	"context"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

type memReportRepo struct {
	m map[string]ports.ReportRecord
}

func (m *memReportRepo) Upsert(_ context.Context, r ports.ReportRecord) error {
	if m.m == nil {
		m.m = map[string]ports.ReportRecord{}
	}
	m.m[r.ID] = r
	return nil
}
func (m *memReportRepo) GetByID(_ context.Context, id string) (ports.ReportRecord, error) {
	r, ok := m.m[id]
	if !ok {
		return ports.ReportRecord{}, contracts.ErrNotFound
	}
	return r, nil
}
func (m *memReportRepo) List(context.Context, int, int) ([]ports.ReportRecord, error) {
	return nil, nil
}
func (m *memReportRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]ports.ReportRecord, error) {
	return nil, nil
}

func TestReport_WithRepositoryReadThrough(t *testing.T) {
	repo := &memReportRepo{}
	s := NewService().WithRepository(repo)
	r, err := s.Create("r1", BuildInput{AssessmentID: "a1", ProjectID: "p1"})
	if err != nil {
		t.Fatal(err)
	}
	s.items = map[string]*Report{}
	got, err := s.Get(r.ID)
	if err != nil || got.SchemaVersion != SchemaVersion {
		t.Fatal(err, got)
	}
	b, err := s.Export(r.ID, contracts.ReportFormatJSON)
	if err != nil || len(b) < 5 {
		t.Fatal(err)
	}
}

package riskprio

import (
	"context"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

type memRiskRepo struct {
	byID map[string]ports.RiskEvaluationRecord
	byF  map[contracts.FindingID]ports.RiskEvaluationRecord
}

func (m *memRiskRepo) Upsert(_ context.Context, r ports.RiskEvaluationRecord) error {
	if m.byID == nil {
		m.byID = map[string]ports.RiskEvaluationRecord{}
		m.byF = map[contracts.FindingID]ports.RiskEvaluationRecord{}
	}
	m.byID[r.ID] = r
	m.byF[r.FindingID] = r
	return nil
}
func (m *memRiskRepo) GetByID(_ context.Context, id string) (ports.RiskEvaluationRecord, error) {
	r, ok := m.byID[id]
	if !ok {
		return ports.RiskEvaluationRecord{}, contracts.ErrNotFound
	}
	return r, nil
}
func (m *memRiskRepo) GetByFinding(_ context.Context, id contracts.FindingID) (ports.RiskEvaluationRecord, error) {
	r, ok := m.byF[id]
	if !ok {
		return ports.RiskEvaluationRecord{}, contracts.ErrNotFound
	}
	return r, nil
}
func (m *memRiskRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]ports.RiskEvaluationRecord, error) {
	return nil, nil
}
func (m *memRiskRepo) ListByAssessment(context.Context, string, int, int) ([]ports.RiskEvaluationRecord, error) {
	return nil, nil
}

func TestEngine_WithRepositoryReadThrough(t *testing.T) {
	repo := &memRiskRepo{}
	e := NewEngine().WithRepository(repo)
	ev, err := e.Evaluate(Input{
		FindingID: "f1", ProjectID: "p1", Severity: contracts.FindingSeverityHigh,
		Confidence: contracts.FindingConfidenceHigh, AssetCriticality: contracts.AssetCriticalityMedium,
		Exposure: contracts.ExposureExternal,
	})
	if err != nil {
		t.Fatal(err)
	}
	e.items = map[string]*Evaluation{}
	e.all = map[string]*Evaluation{}
	got, err := e.GetByFinding("f1")
	if err != nil || got.Score != ev.Score {
		t.Fatal(err, got)
	}
}

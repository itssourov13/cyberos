package evidence

import (
	"context"
	"errors"
	"testing"
	"time"

	"cyberos/internal/contracts"
	domainevidence "cyberos/internal/domain/evidence"
)

type memEvidenceRepo struct {
	m    map[contracts.EvidenceID]*domainevidence.Evidence
	fail bool
}

func (m *memEvidenceRepo) Upsert(_ context.Context, e *domainevidence.Evidence) error {
	if m.fail {
		return errors.New("db down")
	}
	if m.m == nil {
		m.m = map[contracts.EvidenceID]*domainevidence.Evidence{}
	}
	cp := *e
	m.m[e.ID] = &cp
	return nil
}
func (m *memEvidenceRepo) GetByID(_ context.Context, id contracts.EvidenceID) (*domainevidence.Evidence, error) {
	e, ok := m.m[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *e
	return &cp, nil
}
func (m *memEvidenceRepo) ListByTask(context.Context, contracts.TaskID, int, int) ([]*domainevidence.Evidence, error) {
	return nil, nil
}
func (m *memEvidenceRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainevidence.Evidence, error) {
	return nil, nil
}

func testGrant(id, task string) contracts.ExecutionGrant {
	now := time.Now().UTC()
	return contracts.ExecutionGrant{
		ID: contracts.ExecutionGrantID(id), TaskID: contracts.TaskID(task),
		ProjectID: "p1", ScopeID: "s1", CapabilityID: "cap.1",
		Principal: contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman},
		Status:    contracts.ExecutionGrantStatusIssued,
		IssuedAt:  now, ExpiresAt: now.Add(time.Hour),
	}
}

func TestEvidence_WithRepositoryReadThroughAndFail(t *testing.T) {
	repo := &memEvidenceRepo{}
	s := NewService().WithRepository(repo)
	ev, err := s.RecordFromExecution(contracts.ExecutionResult{
		Status: contracts.ExecutionStatusSucceeded, Output: map[string]any{"ok": true},
		CorrelationID: "c1", ExecutorID: "mock",
	}, testGrant("g1", "t1"), "ev-1")
	if err != nil {
		t.Fatal(err)
	}
	s.items = map[contracts.EvidenceID]*domainevidence.Evidence{}
	got, err := s.Get(ev.ID)
	if err != nil || got.ID != ev.ID {
		t.Fatal(err, got)
	}

	s2 := NewService().WithRepository(&memEvidenceRepo{fail: true})
	_, err = s2.RecordFromExecution(contracts.ExecutionResult{
		Status: contracts.ExecutionStatusSucceeded, Output: map[string]any{},
	}, testGrant("g2", "t2"), "ev-fail")
	if err == nil {
		t.Fatal("expected persist error")
	}
	if _, err := s2.Get("ev-fail"); !errors.Is(err, contracts.ErrNotFound) && err == nil {
		t.Fatal("must not keep failed write")
	}
}

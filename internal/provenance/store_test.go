package provenance

import (
	"context"
	"errors"
	"testing"

	"cyberos/internal/contracts"
	domainevidence "cyberos/internal/domain/evidence"
	domainprov "cyberos/internal/domain/provenance"
)

type memProvRepo struct {
	m    map[string]*domainprov.Provenance
	fail bool
}

func (m *memProvRepo) Upsert(_ context.Context, p *domainprov.Provenance) error {
	if m.fail {
		return errors.New("db down")
	}
	if m.m == nil {
		m.m = map[string]*domainprov.Provenance{}
	}
	cp := *p
	m.m[p.SubjectType+":"+p.SubjectID] = &cp
	return nil
}
func (m *memProvRepo) Get(_ context.Context, st, sid string) (*domainprov.Provenance, error) {
	p, ok := m.m[st+":"+sid]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *memProvRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainprov.Provenance, error) {
	return nil, nil
}

func TestProvenance_WithRepositoryReadThroughAndFail(t *testing.T) {
	repo := &memProvRepo{}
	s := NewService().WithRepository(repo)
	ev, err := domainevidence.New("ev1", "t1", "g1", "cap.1",
		contracts.PrincipalRef{ID: "u1", Type: contracts.PrincipalTypeHuman}, "p1", "s1")
	if err != nil {
		t.Fatal(err)
	}
	ev.CorrelationID = "c1"
	p, err := s.RecordForEvidence(ev, "mock")
	if err != nil {
		t.Fatal(err)
	}
	s.items = map[string]*domainprov.Provenance{}
	got, err := s.Get("evidence", string(ev.ID))
	if err != nil || got.SubjectID != p.SubjectID {
		t.Fatal(err, got)
	}

	s2 := NewService().WithRepository(&memProvRepo{fail: true})
	_, err = s2.RecordForEvidence(ev, "mock")
	if err == nil {
		t.Fatal("expected persist error")
	}
}

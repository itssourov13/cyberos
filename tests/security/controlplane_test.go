package security_test

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/application/scope"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/identity"
)

type cpPrincipalRepo struct {
	items map[contracts.PrincipalID]*principal.Principal
}

func (m *cpPrincipalRepo) Create(_ context.Context, p *principal.Principal) error {
	if m.items == nil {
		m.items = map[contracts.PrincipalID]*principal.Principal{}
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}
func (m *cpPrincipalRepo) GetByID(_ context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *cpPrincipalRepo) List(context.Context, int, int) ([]*principal.Principal, error) {
	return nil, nil
}

type cpScopeRepo struct {
	items map[contracts.ScopeID]*domainscope.Scope
}

func (m *cpScopeRepo) Create(_ context.Context, s *domainscope.Scope) error {
	if m.items == nil {
		m.items = map[contracts.ScopeID]*domainscope.Scope{}
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}
func (m *cpScopeRepo) GetByID(_ context.Context, id contracts.ScopeID) (*domainscope.Scope, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (m *cpScopeRepo) Update(context.Context, *domainscope.Scope) error { return nil }
func (m *cpScopeRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}

func newCP(t *testing.T) (*controlplane.Service, *cpPrincipalRepo, *cpScopeRepo) {
	t.Helper()
	pr := &cpPrincipalRepo{}
	sr := &cpScopeRepo{}
	return controlplane.NewService(identity.NewService(pr), scope.NewService(sr)), pr, sr
}

func TestSecurity_ControlPlane_MissingScopeDenied(t *testing.T) {
	cp, pr, _ := newCP(t)
	p, _ := principal.NewHuman("u1", "a")
	_ = pr.Create(context.Background(), p)

	d, _ := cp.Evaluate(context.Background(), controlplane.EvaluateRequest{
		PrincipalID:  "u1",
		CapabilityID: "c1",
	})
	if d.Allowed() {
		t.Fatal("missing scope must be denied")
	}
	if d.Outcome != contracts.DecisionDenied || d.Reason == "" {
		t.Fatalf("structured denial required: %+v", d)
	}
}

func TestSecurity_ControlPlane_ExpiredScopeDenied(t *testing.T) {
	cp, pr, sr := newCP(t)
	p, _ := principal.NewHuman("u1", "a")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("s1", "p1", "e", contracts.ScopeStatusActive)
	past := time.Now().UTC().Add(-time.Minute)
	sc.ExpiresAt = &past
	_ = sr.Create(context.Background(), sc)

	d, _ := cp.Evaluate(context.Background(), controlplane.EvaluateRequest{
		PrincipalID: "u1", ProjectID: "p1", ScopeID: "s1", CapabilityID: "c1",
	})
	if d.Allowed() {
		t.Fatal("expired scope must be denied")
	}
}

func TestSecurity_ControlPlane_InactivePrincipalDenied(t *testing.T) {
	cp, pr, sr := newCP(t)
	p, _ := principal.NewHuman("u1", "a")
	p.Active = false
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("s1", "p1", "m", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)

	d, _ := cp.Evaluate(context.Background(), controlplane.EvaluateRequest{
		PrincipalID: "u1", ProjectID: "p1", ScopeID: "s1", CapabilityID: "c1",
	})
	if d.Allowed() {
		t.Fatal("inactive principal must be denied")
	}
}

func TestSecurity_ControlPlane_DecisionIsStructured(t *testing.T) {
	cp, pr, sr := newCP(t)
	p, _ := principal.NewHuman("u1", "a")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("s1", "p1", "m", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)

	d, _ := cp.Evaluate(context.Background(), controlplane.EvaluateRequest{
		PrincipalID: "u1", ProjectID: "p1", ScopeID: "s1", CapabilityID: "c1",
	})
	if d.ID == "" || d.Reason == "" || !d.Outcome.Valid() {
		t.Fatalf("decision must be structured, got %+v", d)
	}
}

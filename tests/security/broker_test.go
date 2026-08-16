package security_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cyberos/internal/application/scope"
	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/execution/mock"
	"cyberos/internal/identity"
)

type bMemP struct {
	items map[contracts.PrincipalID]*principal.Principal
}

func (m *bMemP) Create(_ context.Context, p *principal.Principal) error {
	if m.items == nil {
		m.items = map[contracts.PrincipalID]*principal.Principal{}
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}
func (m *bMemP) GetByID(_ context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *bMemP) List(context.Context, int, int) ([]*principal.Principal, error) { return nil, nil }

type bMemS struct {
	items map[contracts.ScopeID]*domainscope.Scope
}

func (m *bMemS) Create(_ context.Context, s *domainscope.Scope) error {
	if m.items == nil {
		m.items = map[contracts.ScopeID]*domainscope.Scope{}
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}
func (m *bMemS) GetByID(_ context.Context, id contracts.ScopeID) (*domainscope.Scope, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (m *bMemS) Update(context.Context, *domainscope.Scope) error { return nil }
func (m *bMemS) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}

// Security invariant: no valid grant ⇒ no execution.
func TestSecurity_NoGrantNoExecution(t *testing.T) {
	brk := broker.New(mock.New())
	_, err := brk.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("execution without grant must fail")
	}
}

// Security invariant: denied Control Plane decision cannot produce a grant.
func TestSecurity_DeniedDecisionNoGrant(t *testing.T) {
	pr := &bMemP{}
	sr := &bMemS{}
	p, _ := principal.NewHuman("u1", "a")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("s1", "p1", "m", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)

	cp := controlplane.NewService(identity.NewService(pr), scope.NewService(sr))
	issuer := broker.NewIssuer(time.Minute)

	// Missing capability → denied
	_, d, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "u1", ProjectID: "p1", ScopeID: "s1",
	})
	if err == nil {
		t.Fatal("expected issue failure")
	}
	if d.Allowed() {
		t.Fatal("decision must be denied")
	}
	if !errors.Is(err, contracts.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// Security invariant: only mock executor is available in Phase 1G.
func TestSecurity_OnlyMockExecutor(t *testing.T) {
	ex := mock.New()
	res, err := ex.Execute(contracts.ExecutionRequest{
		Grant: contracts.ExecutionGrant{
			ID: "g", Principal: contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
			ProjectID: "p", ScopeID: "s", TaskID: "t", CapabilityID: "c",
			Status:    contracts.ExecutionGrantStatusIssued,
			IssuedAt:  time.Now().UTC().Add(-time.Minute),
			ExpiresAt: time.Now().UTC().Add(time.Minute),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExecutorID != "mock" {
		t.Errorf("executor = %s", res.ExecutorID)
	}
	if res.Output["mock"] != true {
		t.Error("expected safe mock marker")
	}
}

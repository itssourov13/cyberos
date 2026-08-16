package broker

import (
	"context"
	"errors"
	"testing"
	"time"

	"cyberos/internal/application/scope"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/execution/mock"
	"cyberos/internal/identity"
)

// --- minimal repos ---

type memP struct {
	items map[contracts.PrincipalID]*principal.Principal
}

func (m *memP) Create(_ context.Context, p *principal.Principal) error {
	if m.items == nil {
		m.items = map[contracts.PrincipalID]*principal.Principal{}
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}
func (m *memP) GetByID(_ context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *memP) List(context.Context, int, int) ([]*principal.Principal, error) { return nil, nil }

type memS struct {
	items map[contracts.ScopeID]*domainscope.Scope
}

func (m *memS) Create(_ context.Context, s *domainscope.Scope) error {
	if m.items == nil {
		m.items = map[contracts.ScopeID]*domainscope.Scope{}
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}
func (m *memS) GetByID(_ context.Context, id contracts.ScopeID) (*domainscope.Scope, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (m *memS) Update(context.Context, *domainscope.Scope) error { return nil }
func (m *memS) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}

func setup(t *testing.T) (*Issuer, *Broker, *controlplane.Service) {
	t.Helper()
	pr := &memP{}
	sr := &memS{}
	p, _ := principal.NewHuman("user-1", "alice")
	_ = pr.Create(context.Background(), p)
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	_ = sr.Create(context.Background(), sc)

	cp := controlplane.NewService(identity.NewService(pr), scope.NewService(sr))
	issuer := NewIssuer(5 * time.Minute)
	brk := New(mock.New())
	return issuer, brk, cp
}

func TestIssueAndExecuteHappyPath(t *testing.T) {
	issuer, brk, cp := setup(t)
	g, d, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CapabilityID: "cap-1", TaskID: "task-1",
	})
	if err != nil {
		t.Fatalf("issue: %v (decision=%s %s)", err, d.Outcome, d.Reason)
	}
	if !g.IsUsable(time.Now().UTC()) {
		t.Fatal("grant should be usable")
	}

	res, err := brk.Execute(contracts.ExecutionRequest{Grant: g.ExecutionGrant})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.Status != contracts.ExecutionStatusSucceeded {
		t.Errorf("status = %s", res.Status)
	}
	if res.ExecutorID != "mock" {
		t.Errorf("executor = %s", res.ExecutorID)
	}
	if res.Output["mock"] != true {
		t.Error("expected mock output")
	}
}

func TestNoGrantNoExecution(t *testing.T) {
	_, brk, _ := setup(t)
	_, err := brk.Execute(contracts.ExecutionRequest{})
	if err == nil {
		t.Fatal("empty grant must fail")
	}
}

func TestDeniedDecisionNoGrant(t *testing.T) {
	issuer, _, cp := setup(t)
	// missing capability → denied
	_, d, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
	})
	if err == nil {
		t.Fatal("expected error on denied decision")
	}
	if d.Allowed() {
		t.Fatal("decision should be denied")
	}
}

func TestExpiredGrantRejected(t *testing.T) {
	issuer, brk, cp := setup(t)
	g, _, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CapabilityID: "cap-1", TaskID: "task-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Force expiry
	g.ExpiresAt = time.Now().UTC().Add(-time.Second)
	_, err = brk.Execute(contracts.ExecutionRequest{Grant: g.ExecutionGrant})
	if !errors.Is(err, contracts.ErrGrantExpired) && !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Errorf("expected grant expired/invalid, got %v", err)
	}
}

func TestReplayNonceRejected(t *testing.T) {
	issuer, brk, cp := setup(t)
	g, _, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CapabilityID: "cap-1", TaskID: "task-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	req := contracts.ExecutionRequest{Grant: g.ExecutionGrant}
	if _, err := brk.Execute(req); err != nil {
		t.Fatalf("first exec: %v", err)
	}
	// Replay same nonce
	_, err = brk.Execute(req)
	if !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Errorf("expected ErrGrantInvalid on replay, got %v", err)
	}
}

func TestMismatchedContextRejected(t *testing.T) {
	issuer, brk, cp := setup(t)
	g, _, err := issuer.IssueAfterEvaluate(context.Background(), cp, controlplane.EvaluateRequest{
		PrincipalID: "user-1", ProjectID: "proj-1", ScopeID: "scope-1",
		CapabilityID: "cap-1", TaskID: "task-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = brk.ExecuteWithContext(context.Background(),
		contracts.ExecutionRequest{Grant: g.ExecutionGrant},
		"other-user", "proj-1", "scope-1", "task-1", "cap-1",
	)
	if !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Errorf("expected ErrGrantInvalid, got %v", err)
	}
}

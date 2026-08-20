package security_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"cyberos/internal/application/scope"
	"cyberos/internal/contracts"
	"cyberos/internal/domain/principal"
	domainscope "cyberos/internal/domain/scope"
	"cyberos/internal/identity"
)

// In-memory repos duplicated lightly for security-focused tests
// (avoids importing test helpers across packages).

type memPrincipalRepo struct {
	items map[contracts.PrincipalID]*principal.Principal
}

func (m *memPrincipalRepo) Create(_ context.Context, p *principal.Principal) error {
	if m.items == nil {
		m.items = make(map[contracts.PrincipalID]*principal.Principal)
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}
func (m *memPrincipalRepo) GetByID(_ context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}
func (m *memPrincipalRepo) List(context.Context, int, int) ([]*principal.Principal, error) {
	return nil, nil
}

type memScopeRepo struct {
	items map[contracts.ScopeID]*domainscope.Scope
}

func (m *memScopeRepo) Create(_ context.Context, s *domainscope.Scope) error {
	if m.items == nil {
		m.items = make(map[contracts.ScopeID]*domainscope.Scope)
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}
func (m *memScopeRepo) GetByID(_ context.Context, id contracts.ScopeID) (*domainscope.Scope, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *s
	return &cp, nil
}
func (m *memScopeRepo) Update(context.Context, *domainscope.Scope) error { return nil }
func (m *memScopeRepo) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}
func (m *memScopeRepo) ListAll(context.Context, int, int) ([]*domainscope.Scope, error) {
	return nil, nil
}

// Security invariant: inactive principal cannot pass RequireActive.
func TestSecurity_InactivePrincipalRejected(t *testing.T) {
	repo := &memPrincipalRepo{}
	svc := identity.NewService(repo)
	p, _ := principal.NewHuman("u1", "alice")
	p.Active = false
	_ = repo.Create(context.Background(), p)

	_, err := svc.RequireActive(context.Background(), "u1")
	if !errors.Is(err, contracts.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// Security invariant: expired scope cannot pass RequireActive.
func TestSecurity_ExpiredScopeRejected(t *testing.T) {
	repo := &memScopeRepo{}
	svc := scope.NewService(repo)
	sc, _ := domainscope.New("s1", "p1", "exp", contracts.ScopeStatusActive)
	past := time.Now().UTC().Add(-time.Minute)
	sc.ExpiresAt = &past
	_ = repo.Create(context.Background(), sc)

	_, err := svc.RequireActive(context.Background(), "s1", time.Now().UTC())
	if !errors.Is(err, contracts.ErrScopeInvalid) {
		t.Fatalf("expected ErrScopeInvalid, got %v", err)
	}
}

// Security invariant: out-of-scope asset is forbidden.
func TestSecurity_OutOfScopeAssetRejected(t *testing.T) {
	svc := scope.NewService(&memScopeRepo{})
	sc, _ := domainscope.New("s1", "p1", "restricted", contracts.ScopeStatusActive)
	sc.Constraints.AllowedAssetIDs = []contracts.AssetID{"allowed-only"}

	err := svc.EnsureAllowsAsset(sc, "other-asset")
	if !errors.Is(err, contracts.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

// Security invariant: missing scope is treated as invalid boundary.
func TestSecurity_MissingScopeInvalid(t *testing.T) {
	svc := scope.NewService(&memScopeRepo{})
	_, err := svc.RequireActive(context.Background(), "nope", time.Now().UTC())
	if !errors.Is(err, contracts.ErrScopeInvalid) {
		t.Fatalf("expected ErrScopeInvalid, got %v", err)
	}
}

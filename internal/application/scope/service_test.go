package scope

import (
	"context"
	"errors"
	"testing"
	"time"

	"cyberos/internal/contracts"
	domainscope "cyberos/internal/domain/scope"
)

type memoryScopeRepo struct {
	items map[contracts.ScopeID]*domainscope.Scope
}

func newMemScopeRepo() *memoryScopeRepo {
	return &memoryScopeRepo{items: make(map[contracts.ScopeID]*domainscope.Scope)}
}

func (m *memoryScopeRepo) Create(_ context.Context, s *domainscope.Scope) error {
	if _, ok := m.items[s.ID]; ok {
		return contracts.ErrConflict
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}

func (m *memoryScopeRepo) GetByID(_ context.Context, id contracts.ScopeID) (*domainscope.Scope, error) {
	s, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (m *memoryScopeRepo) Update(_ context.Context, s *domainscope.Scope) error {
	if _, ok := m.items[s.ID]; !ok {
		return contracts.ErrNotFound
	}
	cp := *s
	m.items[s.ID] = &cp
	return nil
}

func (m *memoryScopeRepo) ListByProject(_ context.Context, projectID contracts.ProjectID, limit, offset int) ([]*domainscope.Scope, error) {
	var out []*domainscope.Scope
	for _, s := range m.items {
		if s.ProjectID != projectID {
			continue
		}
		cp := *s
		out = append(out, &cp)
	}
	return out, nil
}

func TestRequireActiveHappy(t *testing.T) {
	repo := newMemScopeRepo()
	svc := NewService(repo)
	ctx := context.Background()

	sc, err := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Create(ctx, sc); err != nil {
		t.Fatal(err)
	}

	got, err := svc.RequireActive(ctx, "scope-1", time.Now().UTC())
	if err != nil {
		t.Fatalf("RequireActive: %v", err)
	}
	if got.ID != "scope-1" {
		t.Error("id mismatch")
	}
}

func TestRequireActiveExpired(t *testing.T) {
	repo := newMemScopeRepo()
	svc := NewService(repo)
	ctx := context.Background()

	sc, _ := domainscope.New("scope-exp", "proj-1", "Expired", contracts.ScopeStatusActive)
	past := time.Now().UTC().Add(-time.Hour)
	sc.ExpiresAt = &past
	_ = svc.Create(ctx, sc)

	_, err := svc.RequireActive(ctx, "scope-exp", time.Now().UTC())
	if err == nil {
		t.Fatal("expected error for expired scope")
	}
	if !errors.Is(err, contracts.ErrScopeInvalid) {
		t.Errorf("expected ErrScopeInvalid, got %v", err)
	}
}

func TestRequireActiveInactiveStatus(t *testing.T) {
	repo := newMemScopeRepo()
	svc := NewService(repo)
	ctx := context.Background()

	sc, _ := domainscope.New("scope-inact", "proj-1", "Inactive", contracts.ScopeStatusInactive)
	_ = svc.Create(ctx, sc)

	_, err := svc.RequireActive(ctx, "scope-inact", time.Now().UTC())
	if !errors.Is(err, contracts.ErrScopeInvalid) {
		t.Errorf("expected ErrScopeInvalid, got %v", err)
	}
}

func TestRequireActiveMissing(t *testing.T) {
	svc := NewService(newMemScopeRepo())
	_, err := svc.RequireActive(context.Background(), "missing", time.Now().UTC())
	if !errors.Is(err, contracts.ErrScopeInvalid) {
		t.Errorf("expected ErrScopeInvalid, got %v", err)
	}
}

func TestEnsureAllowsAsset(t *testing.T) {
	svc := NewService(newMemScopeRepo())
	sc, _ := domainscope.New("scope-1", "proj-1", "Restricted", contracts.ScopeStatusActive)
	sc.Constraints.AllowedAssetIDs = []contracts.AssetID{"a1", "a2"}

	if err := svc.EnsureAllowsAsset(sc, "a1"); err != nil {
		t.Errorf("a1 should be allowed: %v", err)
	}
	err := svc.EnsureAllowsAsset(sc, "a99")
	if !errors.Is(err, contracts.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestEnsureSameProject(t *testing.T) {
	sc, _ := domainscope.New("scope-1", "proj-1", "Main", contracts.ScopeStatusActive)
	if err := EnsureSameProject(sc, "proj-1"); err != nil {
		t.Errorf("same project should pass: %v", err)
	}
	if err := EnsureSameProject(sc, "other"); !errors.Is(err, contracts.ErrScopeInvalid) {
		t.Errorf("expected ErrScopeInvalid, got %v", err)
	}
}

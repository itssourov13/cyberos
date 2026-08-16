package identity

import (
	"context"
	"errors"
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/domain/principal"
)

// memoryPrincipalRepo is a minimal in-memory repo for unit tests.
type memoryPrincipalRepo struct {
	items map[contracts.PrincipalID]*principal.Principal
}

func newMemPrincipalRepo() *memoryPrincipalRepo {
	return &memoryPrincipalRepo{items: make(map[contracts.PrincipalID]*principal.Principal)}
}

func (m *memoryPrincipalRepo) Create(_ context.Context, p *principal.Principal) error {
	if _, ok := m.items[p.ID]; ok {
		return contracts.ErrConflict
	}
	cp := *p
	m.items[p.ID] = &cp
	return nil
}

func (m *memoryPrincipalRepo) GetByID(_ context.Context, id contracts.PrincipalID) (*principal.Principal, error) {
	p, ok := m.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (m *memoryPrincipalRepo) List(_ context.Context, limit, offset int) ([]*principal.Principal, error) {
	var out []*principal.Principal
	i := 0
	for _, p := range m.items {
		if i < offset {
			i++
			continue
		}
		if limit > 0 && len(out) >= limit {
			break
		}
		cp := *p
		out = append(out, &cp)
		i++
	}
	return out, nil
}

func TestCreateAndRequireActive(t *testing.T) {
	repo := newMemPrincipalRepo()
	svc := NewService(repo)
	ctx := context.Background()

	p, err := principal.NewHuman("user-1", "alice")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Create(ctx, p); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := svc.RequireActive(ctx, "user-1")
	if err != nil {
		t.Fatalf("RequireActive: %v", err)
	}
	if got.Name != "alice" {
		t.Errorf("name = %s", got.Name)
	}
}

func TestRequireActiveInactive(t *testing.T) {
	repo := newMemPrincipalRepo()
	svc := NewService(repo)
	ctx := context.Background()

	p, _ := principal.NewHuman("user-2", "bob")
	p.Active = false
	// Create bypasses active check only if we insert directly for this test
	_ = repo.Create(ctx, p)

	_, err := svc.RequireActive(ctx, "user-2")
	if err == nil {
		t.Fatal("expected error for inactive principal")
	}
	if !errors.Is(err, contracts.ErrForbidden) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

func TestRequireActiveMissing(t *testing.T) {
	svc := NewService(newMemPrincipalRepo())
	_, err := svc.RequireActive(context.Background(), "missing")
	if !errors.Is(err, contracts.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCreateInactiveRejected(t *testing.T) {
	svc := NewService(newMemPrincipalRepo())
	p, _ := principal.NewHuman("user-3", "carol")
	p.Active = false
	err := svc.Create(context.Background(), p)
	if err == nil {
		t.Fatal("creating inactive principal should fail")
	}
}

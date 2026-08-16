package grant

import (
	"errors"
	"testing"
	"time"

	"cyberos/internal/contracts"
)

func makeGrant(t *testing.T, expiresOffset time.Duration) *Grant {
	t.Helper()
	now := time.Now().UTC()
	g := contracts.ExecutionGrant{
		ID:           "grant-1",
		Principal:    contracts.PrincipalRef{ID: "user-1", Type: contracts.PrincipalTypeHuman},
		ProjectID:    "proj-1",
		ScopeID:      "scope-1",
		TaskID:       "task-1",
		CapabilityID: "cap-1",
		Status:       contracts.ExecutionGrantStatusIssued,
		IssuedAt:     now.Add(-time.Minute),
		ExpiresAt:    now.Add(expiresOffset),
	}
	dg, err := NewFromContract(g)
	if err != nil {
		t.Fatalf("NewFromContract failed: %v", err)
	}
	return dg
}

func TestUsableGrant(t *testing.T) {
	g := makeGrant(t, 5*time.Minute)
	now := time.Now().UTC()
	if !g.IsUsable(now) {
		t.Error("fresh grant should be usable")
	}
	err := g.EnsureUsable(now, "user-1", "proj-1", "scope-1", "task-1", "cap-1")
	if err != nil {
		t.Errorf("EnsureUsable failed: %v", err)
	}
}

func TestExpiredGrant(t *testing.T) {
	g := makeGrant(t, -time.Second)
	now := time.Now().UTC()
	if g.IsUsable(now) {
		t.Error("expired grant must not be usable")
	}
	err := g.EnsureUsable(now, "user-1", "proj-1", "scope-1", "task-1", "cap-1")
	if !errors.Is(err, contracts.ErrGrantExpired) {
		t.Errorf("expected ErrGrantExpired, got %v", err)
	}
}

func TestMismatchedContext(t *testing.T) {
	g := makeGrant(t, 5*time.Minute)
	now := time.Now().UTC()
	err := g.EnsureUsable(now, "other-user", "proj-1", "scope-1", "task-1", "cap-1")
	if !errors.Is(err, contracts.ErrGrantInvalid) {
		t.Errorf("expected ErrGrantInvalid for principal mismatch, got %v", err)
	}
}

func TestMatchesContext(t *testing.T) {
	g := makeGrant(t, 5*time.Minute)
	if !g.MatchesContext("user-1", "proj-1", "scope-1", "task-1", "cap-1") {
		t.Error("should match")
	}
	if g.MatchesContext("user-1", "proj-1", "scope-1", "task-1", "other-cap") {
		t.Error("capability mismatch should fail")
	}
}

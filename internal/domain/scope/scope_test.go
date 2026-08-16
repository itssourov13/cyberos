package scope

import (
	"testing"
	"time"

	"cyberos/internal/contracts"
)

func TestNewAndIsActive(t *testing.T) {
	s, err := New("scope-1", "proj-1", "Default Scope", contracts.ScopeStatusActive)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	now := time.Now().UTC()
	if !s.IsActive(now) {
		t.Error("active scope with no bounds should be active")
	}
}

func TestExpiredByTime(t *testing.T) {
	s, _ := New("scope-1", "proj-1", "Expired", contracts.ScopeStatusActive)
	past := time.Now().UTC().Add(-time.Hour)
	s.ExpiresAt = &past
	if s.IsActive(time.Now().UTC()) {
		t.Error("scope past expires_at must not be active")
	}
}

func TestInactiveStatus(t *testing.T) {
	s, _ := New("scope-1", "proj-1", "Inactive", contracts.ScopeStatusInactive)
	if s.IsActive(time.Now().UTC()) {
		t.Error("inactive status must not be active")
	}
}

func TestAllowsAsset(t *testing.T) {
	s, _ := New("scope-1", "proj-1", "Scoped", contracts.ScopeStatusActive)
	// No restrictions
	if !s.AllowsAsset("any") {
		t.Error("empty allow list should allow any")
	}
	s.Constraints.AllowedAssetIDs = []contracts.AssetID{"a1", "a2"}
	if !s.AllowsAsset("a1") {
		t.Error("a1 should be allowed")
	}
	if s.AllowsAsset("a3") {
		t.Error("a3 should not be allowed")
	}
}

func TestInvalidTimeWindow(t *testing.T) {
	s, _ := New("scope-1", "proj-1", "Bad", contracts.ScopeStatusActive)
	start := time.Now().UTC()
	end := start.Add(-time.Hour)
	s.StartsAt = &start
	s.ExpiresAt = &end
	if err := s.Validate(); err == nil {
		t.Error("expires before starts should fail validation")
	}
}

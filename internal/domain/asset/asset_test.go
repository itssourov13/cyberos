package asset

import (
	"testing"

	"cyberos/internal/contracts"
)

func TestNewAndValidate(t *testing.T) {
	a, err := New("asset-1", "proj-1", contracts.AssetTypeDomain, "example.com", "example.com")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := a.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !a.BelongsTo("proj-1") {
		t.Error("should belong to proj-1")
	}
	if a.BelongsTo("other") {
		t.Error("should not belong to other")
	}
}

func TestNewInvalidType(t *testing.T) {
	if _, err := New("a1", "p1", contracts.AssetType("BAD"), "n", "v"); err == nil {
		t.Error("bad type should fail")
	}
}

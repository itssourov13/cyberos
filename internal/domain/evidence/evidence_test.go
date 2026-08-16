package evidence

import (
	"testing"

	"cyberos/internal/contracts"
)

func TestNewAndValidate(t *testing.T) {
	e, err := New(
		"ev-1", "task-1", "grant-1", "cap-1",
		contracts.PrincipalRef{ID: "user-1", Type: contracts.PrincipalTypeHuman},
		"proj-1", "scope-1",
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := e.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if e.Ref().ID != "ev-1" {
		t.Error("Ref mismatch")
	}
	meta := e.MetadataView()
	if meta.TaskID != "task-1" || meta.GrantID != "grant-1" {
		t.Error("metadata view mismatch")
	}
}

func TestMissingGrant(t *testing.T) {
	_, err := New("ev-1", "task-1", "", "cap-1",
		contracts.PrincipalRef{ID: "u", Type: contracts.PrincipalTypeHuman},
		"p", "s")
	if err == nil {
		t.Error("missing grant_id should fail")
	}
}

package project

import (
	"testing"
)

func TestNewAndValidate(t *testing.T) {
	p, err := New("proj-1", "Test Project")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !p.IsActive() {
		t.Error("new project should be active")
	}
	if p.Ref().ID != "proj-1" {
		t.Error("Ref ID mismatch")
	}
}

func TestNewInvalid(t *testing.T) {
	if _, err := New("", "name"); err == nil {
		t.Error("empty id should fail")
	}
	if _, err := New("id", ""); err == nil {
		t.Error("empty name should fail")
	}
}

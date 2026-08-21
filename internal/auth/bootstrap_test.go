package auth

import (
	"context"
	"strings"
	"testing"
)

func TestEnsureBootstrap_ReusesExisting(t *testing.T) {
	store := NewMemoryKeyStore()
	a := NewAPIKeyAuthenticator(store)
	plain, _, err := a.Issue(context.Background(), "bootstrap", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	p2, created, err := a.EnsureBootstrap(context.Background(), "", "")
	if err != nil || created || p2 != "" {
		t.Fatalf("should reuse: created=%v plain=%q err=%v", created, p2, err)
	}
	// second call still no re-issue
	_, created, err = a.EnsureBootstrap(context.Background(), "", "")
	if err != nil || created {
		t.Fatal(err, created)
	}
	_ = plain
}

func TestEnsureBootstrap_EnvKey(t *testing.T) {
	store := NewMemoryKeyStore()
	a := NewAPIKeyAuthenticator(store)
	// Create a real key then use its plaintext as env bootstrap on fresh store
	a2 := NewAPIKeyAuthenticator(NewMemoryKeyStore())
	plain, _, err := a2.Issue(context.Background(), "bootstrap", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	_, created, err := a.EnsureBootstrap(context.Background(), plain, "ten-1")
	if err != nil || created {
		t.Fatal(err, created)
	}
	if _, err := a.Authenticate(context.Background(), plain); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(plain, "cos_") {
		t.Fatal(plain)
	}
}

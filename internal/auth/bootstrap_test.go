package auth

import (
	"context"
	"testing"
	"time"
)

func TestBootstrapRejectsTenantScopedCredential(t *testing.T) {
	store := NewMemoryKeyStore()
	authenticator := NewAPIKeyAuthenticator(store)
	tenantKey, _, err := authenticator.IssueWithRole(context.Background(), "user-1", "tenant-a", RoleAdmin, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := authenticator.EnsureBootstrap(context.Background(), tenantKey, ""); err == nil {
		t.Fatal("tenant admin key must not be accepted as platform bootstrap")
	}
}

func TestBootstrapCreatesExplicitOperatorCredential(t *testing.T) {
	store := NewMemoryKeyStore()
	authenticator := NewAPIKeyAuthenticator(store)
	plain, created, err := authenticator.EnsureBootstrap(context.Background(), "", "")
	if err != nil || !created || plain == "" {
		t.Fatalf("bootstrap create: created=%v secret_empty=%v err=%v", created, plain == "", err)
	}
	identity, err := authenticator.Authenticate(context.Background(), plain)
	if err != nil {
		t.Fatal(err)
	}
	if identity.Role != RoleOperator || identity.TenantID != "" || identity.Principal.ID != "bootstrap" {
		t.Fatalf("incorrect bootstrap identity: %+v", identity)
	}
	if _, _, err := authenticator.EnsureBootstrap(context.Background(), plain, ""); err != nil {
		t.Fatal(err)
	}
}

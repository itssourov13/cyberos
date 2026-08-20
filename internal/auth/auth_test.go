package auth

import (
	"context"
	"testing"
	"time"

	"cyberos/internal/contracts"
)

func TestAPIKeyIssueAndAuthenticate(t *testing.T) {
	store := NewMemoryKeyStore()
	a := NewAPIKeyAuthenticator(store)
	plain, key, err := a.Issue(context.Background(), "user-1", "tenant-1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if key.Prefix == "" || plain == "" {
		t.Fatal(key, plain)
	}
	id, err := a.Authenticate(context.Background(), "Bearer "+plain)
	if err != nil {
		t.Fatal(err)
	}
	if id.Principal.ID != "user-1" || id.TenantID != "tenant-1" {
		t.Fatalf("%+v", id)
	}
}

func TestAPIKeyExpiredRejected(t *testing.T) {
	store := NewMemoryKeyStore()
	a := NewAPIKeyAuthenticator(store)
	now := time.Now().UTC()
	a.Now = func() time.Time { return now }
	plain, _, err := a.Issue(context.Background(), "u", "t", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	a.Now = func() time.Time { return now.Add(2 * time.Second) }
	_, err = a.Authenticate(context.Background(), plain)
	if err == nil {
		t.Fatal("expired must fail")
	}
}

func TestAPIKeyRevokedRejected(t *testing.T) {
	store := NewMemoryKeyStore()
	a := NewAPIKeyAuthenticator(store)
	plain, key, _ := a.Issue(context.Background(), "u", "t", 0)
	_ = store.Revoke(context.Background(), key.ID)
	_, err := a.Authenticate(context.Background(), plain)
	if err == nil {
		t.Fatal("revoked must fail")
	}
}

func TestInvalidCredential(t *testing.T) {
	a := NewAPIKeyAuthenticator(NewMemoryKeyStore())
	_, err := a.Authenticate(context.Background(), "Bearer garbage")
	if err == nil {
		t.Fatal("invalid")
	}
}

func TestTenantIsolation(t *testing.T) {
	if err := AssertSameTenant("t1", "t1"); err != nil {
		t.Fatal(err)
	}
	if err := AssertSameTenant("t1", "t2"); err == nil {
		t.Fatal("cross-tenant")
	}
	if err := AssertSameTenant("", ""); err != nil {
		t.Fatal("legacy")
	}
	_ = contracts.TenantStatusActive
}

func TestContextIdentity(t *testing.T) {
	ctx := WithIdentity(context.Background(), Identity{
		Principal: contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman},
		TenantID:  "ten-1",
	})
	p, ok := PrincipalFromContext(ctx)
	if !ok || p.ID != "p1" {
		t.Fatal(p, ok)
	}
	tid, ok := TenantFromContext(ctx)
	if !ok || tid != "ten-1" {
		t.Fatal(tid)
	}
}

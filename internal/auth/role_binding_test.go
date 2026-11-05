package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"cyberos/internal/contracts"
	"time"
)

func TestAuthenticateRejectsInvalidStoredTenantRoleBinding(t *testing.T) {
	for _, tc := range []struct {
		name     string
		tenantID string
		role     Role
	}{
		{name: "tenant scoped operator", tenantID: "tenant-a", role: RoleOperator},
		{name: "tenantless analyst", tenantID: "", role: RoleAnalyst},
		{name: "unknown tenant role", tenantID: "tenant-a", role: "superuser"},
		{name: "missing tenant role", tenantID: "tenant-a", role: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			secret := "cos_testcredentialvalue"
			sum := sha256.Sum256([]byte(secret))
			store := NewMemoryKeyStore()
			key := APIKey{
				ID: "key-" + tc.name, Prefix: secret[:12], Hash: hex.EncodeToString(sum[:]),
				PrincipalID: "p1", TenantID: contracts.TenantID(tc.tenantID),
				Role: tc.role, Active: true, CreatedAt: time.Now().UTC(),
			}
			if err := store.Create(context.Background(), key); err != nil {
				t.Fatal(err)
			}
			if _, err := NewAPIKeyAuthenticator(store).Authenticate(context.Background(), secret); err == nil {
				t.Fatal("invalid stored role/tenant binding must fail closed")
			}
		})
	}
}

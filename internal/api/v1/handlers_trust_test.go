package v1

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cyberos/internal/audit"
	"cyberos/internal/auth"
	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/sdk"
	"cyberos/internal/tools"
)

func trustHarness(t *testing.T, role auth.Role, tenant string) (*Server, string) {
	t.Helper()
	keys := auth.NewMemoryKeyStore()
	authenticator := auth.NewAPIKeyAuthenticator(keys)
	secret, _, err := authenticator.IssueWithRole(context.Background(), "bootstrap", contracts.TenantID(tenant), role, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	client := testClient(t)
	sharedAudit := audit.NewService(nil)
	client.Audit = sharedAudit
	client.Catalog.UpsertProject(sdk.ProjectView{
		ID: "project-1", TenantID: "tenant-a", Name: "Test project", CreatedAt: time.Now().UTC(),
	})
	srv := NewServer(client)
	srv.Auth = auth.NewMiddleware(authenticator)
	srv.TrustAdmin = &tools.TrustAdmin{
		Store: tools.NewTrustStore(),
		Keys:  tools.NewSignerKeyRegistry(),
		Audit: sharedAudit,
	}
	return srv, secret
}

func performJSON(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestTrustAdminHTTPFlowsAreOperatorOnlyAndAudited(t *testing.T) {
	srv, operatorKey := trustHarness(t, auth.RoleOperator, "")
	h := srv.Handler()
	digest := tools.DigestPrefix + strings.Repeat("a", 64)

	trusted := performJSON(t, h, http.MethodPost, "/api/v1/trust/packs/trust", operatorKey, map[string]any{
		"project_id": "project-1", "digest": digest, "reason": "reviewed pack", "correlation_id": "trust-corr-1",
	})
	if trusted.Code != http.StatusOK {
		t.Fatalf("trust pack: %d %s", trusted.Code, trusted.Body.String())
	}
	state := performJSON(t, h, http.MethodGet, "/api/v1/trust/packs?project_id=project-1&digest="+digest, operatorKey, nil)
	if state.Code != http.StatusOK || !strings.Contains(state.Body.String(), "\"trusted\":true") {
		t.Fatalf("trust status: %d %s", state.Code, state.Body.String())
	}

	revoked := performJSON(t, h, http.MethodPost, "/api/v1/trust/packs/revoke", operatorKey, map[string]any{
		"project_id": "project-1", "digest": digest, "reason": "pack superseded", "correlation_id": "trust-corr-2",
	})
	if revoked.Code != http.StatusOK {
		t.Fatalf("revoke pack: %d %s", revoked.Code, revoked.Body.String())
	}
	state = performJSON(t, h, http.MethodGet, "/api/v1/trust/packs?project_id=project-1&digest="+digest, operatorKey, nil)
	if state.Code != http.StatusOK || !strings.Contains(state.Body.String(), "\"revoked\":true") {
		t.Fatalf("revoked trust status: %d %s", state.Code, state.Body.String())
	}

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	registered := performJSON(t, h, http.MethodPost, "/api/v1/trust/signers", operatorKey, map[string]any{
		"key_id": "unit-test-signer", "algorithm": "ed25519",
		"public_key": base64.StdEncoding.EncodeToString(pub), "correlation_id": "signer-corr-1",
	})
	if registered.Code != http.StatusCreated {
		t.Fatalf("register signer: %d %s", registered.Code, registered.Body.String())
	}
	disabled := performJSON(t, h, http.MethodPost, "/api/v1/trust/signers/unit-test-signer/disable", operatorKey, nil)
	if disabled.Code != http.StatusOK || !strings.Contains(disabled.Body.String(), "DISABLED") {
		t.Fatalf("disable signer: %d %s", disabled.Code, disabled.Body.String())
	}
	revokedSigner := performJSON(t, h, http.MethodPost, "/api/v1/trust/signers/unit-test-signer/revoke", operatorKey, nil)
	if revokedSigner.Code != http.StatusOK || !strings.Contains(revokedSigner.Body.String(), "REVOKED") {
		t.Fatalf("revoke signer: %d %s", revokedSigner.Code, revokedSigner.Body.String())
	}
	entries, err := srv.SDK.Audit.List("", 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 4 {
		t.Fatalf("expected audit entries for pack/signer changes, got %d", len(entries))
	}
}

func TestTenantAdministratorCannotMutatePlatformTrust(t *testing.T) {
	srv, tenantKey := trustHarness(t, auth.RoleAdmin, "tenant-a")
	digest := tools.DigestPrefix + strings.Repeat("b", 64)
	rr := performJSON(t, srv.Handler(), http.MethodPost, "/api/v1/trust/packs/trust", tenantKey, map[string]any{
		"project_id": "project-1", "digest": digest, "reason": "tenant attempt",
	})
	if rr.Code != http.StatusForbidden {
		t.Fatalf("tenant administrator trust mutation: %d %s", rr.Code, rr.Body.String())
	}
}

func TestDurableGrantRevocationEndpointAuditsAndIsIdempotent(t *testing.T) {
	srv, operatorKey := trustHarness(t, auth.RoleOperator, "")
	now := time.Now().UTC()
	ledger := srv.SDK.Broker.Ledger
	err := ledger.Record(context.Background(), broker.GrantRecord{
		GrantID: "grant-revoke-test", Nonce: "nonce-revoke-test",
		PrincipalID: "tenant-user", ProjectID: "project-1", ScopeID: "scope-1",
		TaskID: "task-1", CapabilityID: "cap-1", CorrelationID: "grant-corr-1",
		IssuedAt: now, ExpiresAt: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	revoked := performJSON(t, h, http.MethodPost, "/api/v1/grants/grant-revoke-test/revoke", operatorKey, map[string]any{
		"reason": "operator cancelled before execution",
	})
	if revoked.Code != http.StatusOK {
		t.Fatalf("revoke grant: %d %s", revoked.Code, revoked.Body.String())
	}
	record, err := ledger.Get(context.Background(), "grant-revoke-test")
	if err != nil || record.State != broker.GrantStateRevoked {
		t.Fatalf("grant ledger state=%q err=%v", record.State, err)
	}
	entries, err := srv.SDK.Audit.List("project-1", 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.Action == "grant.revoked" && entry.GrantID == "grant-revoke-test" {
			found = true
		}
	}
	if !found {
		t.Fatal("grant revocation was not written to the audit log")
	}
	again := performJSON(t, h, http.MethodPost, "/api/v1/grants/grant-revoke-test/revoke", operatorKey, nil)
	if again.Code != http.StatusOK || !strings.Contains(again.Body.String(), "\"already_revoked\":true") {
		t.Fatalf("repeat revocation should be idempotent: %d %s", again.Code, again.Body.String())
	}
}

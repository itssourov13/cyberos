package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

func TestOperatorCanIssueAndRevokeTenantAPIKeys(t *testing.T) {
	client := testClient(t)
	store := auth.NewMemoryKeyStore()
	authenticator := auth.NewAPIKeyAuthenticator(store)
	operatorKey, _, err := authenticator.IssueWithRole(context.Background(), "user-1", "", auth.RoleOperator, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(client)
	srv.Auth = auth.NewMiddleware(authenticator)
	srv.Auth.Sessions = auth.NewSessionStore(time.Hour, time.Hour)
	h := srv.Handler()

	body := map[string]any{
		"principal_id": "user-1",
		"tenant_id":    "tenant-a",
		"role":         "viewer",
		"ttl_seconds":  3600,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/keys", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer "+operatorKey)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("issue key: %d %s", rr.Code, rr.Body.String())
	}
	var issued struct {
		ID        string `json:"id"`
		APIKey    string `json:"api_key"`
		Role      string `json:"role"`
		ShownOnce bool   `json:"shown_once"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &issued); err != nil {
		t.Fatal(err)
	}
	if issued.ID == "" || issued.APIKey == "" || issued.Role != "viewer" || !issued.ShownOnce {
		t.Fatalf("incomplete key issue response: %+v", issued)
	}
	identity, err := authenticator.Authenticate(context.Background(), issued.APIKey)
	if err != nil || identity.Role != auth.RoleViewer || identity.TenantID != "tenant-a" {
		t.Fatalf("issued key identity mismatch: %+v %v", identity, err)
	}

	revoke := httptest.NewRequest(http.MethodPost, "/api/v1/auth/keys/"+issued.ID+"/revoke", nil)
	revoke.Header.Set("Authorization", "Bearer "+operatorKey)
	revoked := httptest.NewRecorder()
	h.ServeHTTP(revoked, revoke)
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke key: %d %s", revoked.Code, revoked.Body.String())
	}
	if _, err := authenticator.Authenticate(context.Background(), issued.APIKey); err == nil {
		t.Fatal("revoked API key still authenticates")
	}
}

func TestTenantViewerCannotAdministerAPIKeys(t *testing.T) {
	client := testClient(t)
	store := auth.NewMemoryKeyStore()
	authenticator := auth.NewAPIKeyAuthenticator(store)
	viewerKey, _, err := authenticator.IssueWithRole(context.Background(), "user-1", "tenant-a", auth.RoleViewer, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(client)
	srv.Auth = auth.NewMiddleware(authenticator)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/keys", bytes.NewBufferString(`{"principal_id":"user-1","tenant_id":"tenant-a","role":"analyst","ttl_seconds":3600}`))
	req.Header.Set("Authorization", "Bearer "+viewerKey)
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer API key admin attempt: %d %s", rr.Code, rr.Body.String())
	}
}

func TestRoleFromContextFailsClosedForMissingTenantRole(t *testing.T) {
	ctx := auth.WithIdentity(context.Background(), auth.Identity{
		Principal: contracts.PrincipalRef{ID: "p1", Type: contracts.PrincipalTypeHuman},
		TenantID:  "tenant-a",
	})
	if role := auth.RoleFromContext(ctx); role.Allows(auth.PermWrite) {
		t.Fatalf("missing tenant role must not be upgraded to analyst: %q", role)
	}
}

package security_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "cyberos/internal/api/v1"
	"cyberos/internal/auth"
	"cyberos/internal/contracts"
	"cyberos/internal/sdk"
)

func TestSecurity_AuthDoesNotAuthorizeExecution(t *testing.T) {
	store := auth.NewMemoryKeyStore()
	a := auth.NewAPIKeyAuthenticator(store)
	plain, _, err := a.Issue(context.Background(), "user-1", "tenant-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	id, err := a.Authenticate(context.Background(), plain)
	if err != nil {
		t.Fatal(err)
	}
	// Authenticated but no grant — broker still requires grant
	client := sdk.NewInMemory()
	_, err = client.Execute(contracts.ExecutionGrant{}, nil)
	if err == nil {
		t.Fatal("auth must not enable execution without grant")
	}
	_ = id
}

func TestSecurity_APIRequiresAuthWhenConfigured(t *testing.T) {
	store := auth.NewMemoryKeyStore()
	a := auth.NewAPIKeyAuthenticator(store)
	plain, _, _ := a.Issue(context.Background(), "user-1", "tenant-a", time.Hour)
	srv := v1.NewServer(sdk.NewInMemory())
	srv.Auth = auth.NewMiddleware(a)
	h := srv.Handler()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 with key, got %d %s", rr.Code, rr.Body.String())
	}

	// health remains public
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health: %d", rr.Code)
	}
}

func TestSecurity_CrossTenantRejected(t *testing.T) {
	if err := auth.AssertSameTenant("tenant-a", "tenant-b"); err == nil {
		t.Fatal("must reject")
	}
}

func TestSecurity_SSERequiresTokenWhenAuthOn(t *testing.T) {
	a := auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore())
	plain, _, _ := a.Issue(context.Background(), "u", "t", 0)
	srv := v1.NewServer(sdk.NewInMemory())
	srv.Auth = auth.NewMiddleware(a)
	h := srv.Handler()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("sse without token: %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream?access_token="+plain, nil)
	h.ServeHTTP(rr, req)
	// stream may hang or return 200 — not 401
	if rr.Code == http.StatusUnauthorized {
		t.Fatal("sse with token must authenticate")
	}
}

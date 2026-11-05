package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cyberos/internal/auth"
	"cyberos/internal/sdk"
)

func TestCORSPreflightAllowsOnlyConfiguredOrigin(t *testing.T) {
	srv := NewServer(sdk.NewInMemory())
	authenticator := auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore())
	srv.Auth = auth.NewMiddleware(authenticator)
	srv.Auth.AllowedOrigins = []string{"https://app.example.test"}
	h := srv.Handler()

	allowed := httptest.NewRequest(http.MethodOptions, "/api/v1/projects", nil)
	allowed.Header.Set("Origin", "https://app.example.test")
	allowed.Header.Set("Access-Control-Request-Method", "POST")
	allowed.Header.Set("Access-Control-Request-Headers", "content-type,x-csrf-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, allowed)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("allowed preflight: %d %s", rr.Code, rr.Body.String())
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://app.example.test" ||
		rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("missing credentialed CORS headers: %v", rr.Header())
	}

	denied := httptest.NewRequest(http.MethodOptions, "/api/v1/projects", nil)
	denied.Header.Set("Origin", "https://evil.example.test")
	denied.Header.Set("Access-Control-Request-Method", "POST")
	blocked := httptest.NewRecorder()
	h.ServeHTTP(blocked, denied)
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("unconfigured preflight must be rejected: %d", blocked.Code)
	}
}

func TestOriginAllowedRequiresExactSchemeAndHost(t *testing.T) {
	m := auth.NewMiddleware(auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore()))
	m.AllowedOrigins = []string{"https://app.example.test"}
	req := httptest.NewRequest(http.MethodPost, "http://api.example.test/api/v1/cases", nil)
	req.Host = "api.example.test"
	req.Header.Set("Origin", "http://api.example.test")
	if !m.OriginAllowed(req) {
		t.Fatal("exact same-origin request should pass")
	}
	req.Header.Set("Origin", "https://api.example.test")
	if m.OriginAllowed(req) {
		t.Fatal("scheme downgrade/mismatch must not be treated as same-origin")
	}
	req.Header.Set("Origin", "https://evilapp.example.test")
	if m.OriginAllowed(req) {
		t.Fatal("lookalike host must not pass origin validation")
	}
}

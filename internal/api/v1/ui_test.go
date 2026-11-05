package v1

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func spaFixture() http.Handler {
	return newSPAHandler(fstest.MapFS{
		"index.html":        {Data: []byte(`<html><script>var t=1</script><script type="module" src="/assets/app-abc.js"></script></html>`)},
		"assets/app-abc.js": {Data: []byte("console.log(1)")},
		"favicon.svg":       {Data: []byte("<svg/>")},
		"PLACEHOLDER":       {Data: []byte("x")},
	})
}

func get(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(method, target, nil))
	return rr
}

func TestSPA_RoutesAssetsAndFallback(t *testing.T) {
	h := spaFixture()
	if rr := get(h, "GET", "/app/projects/p1"); rr.Code != 200 || !strings.Contains(rr.Body.String(), "<html>") {
		t.Fatalf("client route must fall back to index.html: %d", rr.Code)
	}
	rr := get(h, "GET", "/assets/app-abc.js")
	if rr.Code != 200 || !strings.Contains(rr.Header().Get("Content-Type"), "javascript") ||
		!strings.Contains(rr.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("asset: %d %v", rr.Code, rr.Header())
	}
	if rr := get(h, "GET", "/"); rr.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("index must revalidate: %v", rr.Header())
	}
	if rr := get(h, "GET", "/assets/missing.js"); rr.Code != 404 {
		t.Fatalf("missing asset must 404, not index.html: %d", rr.Code)
	}
	if rr := get(h, "GET", "/PLACEHOLDER"); rr.Body.String() == "x" {
		t.Fatal("placeholder file content must not be served")
	}
}

func TestSPA_APIPathsNeverReturnIndex(t *testing.T) {
	h := spaFixture()
	for _, p := range []string{"/api/v1/nope", "/api/unknown", "/api"} {
		rr := get(h, "GET", p)
		if rr.Code != 404 || strings.Contains(rr.Body.String(), "<html>") ||
			!strings.Contains(rr.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("%s: %d %q", p, rr.Code, rr.Body.String())
		}
	}
}

func TestSPA_TraversalAndMethods(t *testing.T) {
	h := spaFixture()
	for _, p := range []string{"/../etc/passwd", "/..%2f..%2fetc/passwd", "/assets/../../x", "/%2e%2e/secret"} {
		rr := get(h, "GET", p)
		if strings.Contains(rr.Body.String(), "root:") || rr.Code >= 500 {
			t.Fatalf("%s: unsafe response %d", p, rr.Code)
		}
	}
	if rr := get(h, "POST", "/"); rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST must be 405: %d", rr.Code)
	}
}

func TestSPA_CSPHashesInlineBootstrapOnly(t *testing.T) {
	rr := get(spaFixture(), "GET", "/")
	csp := rr.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "'sha256-") || strings.Contains(csp, "script-src 'self' 'unsafe-inline'") ||
		!strings.Contains(csp, "frame-ancestors 'none'") || rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("csp/headers: %q %v", csp, rr.Header())
	}
}

func TestSPA_NotBuiltIs503ForRoutes(t *testing.T) {
	h := newSPAHandler(fstest.MapFS{"PLACEHOLDER": {Data: []byte("x")}})
	if rr := get(h, "GET", "/"); rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("unbuilt frontend must be 503, got %d", rr.Code)
	}
}

func TestSPA_MountedAfterAPIRoutes(t *testing.T) {
	srv := NewServer(nil)
	rr := get(srv.Handler(), "GET", "/api/v1/health")
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), `"ok"`) {
		t.Fatalf("health: %d %s", rr.Code, rr.Body.String())
	}
	if rr := get(srv.Handler(), "GET", "/api/v1/does-not-exist"); rr.Code != 404 || strings.Contains(rr.Body.String(), "<html") {
		t.Fatalf("unknown API route: %d", rr.Code)
	}
}

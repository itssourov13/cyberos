package security_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	v1 "cyberos/internal/api/v1"
	"cyberos/internal/auth"
	"cyberos/internal/sdk"
)

type sessionHarness struct {
	h     http.Handler
	plain string
	store *auth.SessionStore
}

func newSessionHarness(t *testing.T) *sessionHarness {
	t.Helper()
	a := auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore())
	plain, _, err := a.Issue(context.Background(), "user-1", "tenant-a", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	srv := v1.NewServer(sdk.NewInMemory())
	srv.Auth = auth.NewMiddleware(a)
	srv.Auth.Sessions = auth.NewSessionStore(time.Hour, time.Hour)
	return &sessionHarness{h: srv.Handler(), plain: plain, store: srv.Auth.Sessions}
}

func (s *sessionHarness) do(r *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.h.ServeHTTP(rr, r)
	return rr
}

func (s *sessionHarness) login(t *testing.T, key string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"api_key":"`+key+`"}`))
	req.Host = "cyberos.test"
	req.Header.Set("Origin", "http://cyberos.test")
	return s.do(req)
}

func sessionCookie(rr *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == auth.SessionCookieName {
			return c
		}
	}
	return nil
}

func TestSession_LoginBadKeyRejectedNoCookie(t *testing.T) {
	s := newSessionHarness(t)
	rr := s.login(t, "cos_notarealkey000000000000")
	if rr.Code != http.StatusUnauthorized || sessionCookie(rr) != nil {
		t.Fatalf("want 401 without cookie, got %d", rr.Code)
	}
}

func TestSession_LoginCookieAttributesAndMe(t *testing.T) {
	s := newSessionHarness(t)
	rr := s.login(t, s.plain)
	if rr.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rr.Code, rr.Body.String())
	}
	c := sessionCookie(rr)
	if c == nil || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie must be HttpOnly+SameSite=Strict: %+v", c)
	}
	if bytes.Contains(rr.Body.Bytes(), []byte(s.plain)) {
		t.Fatal("login response must never echo the API key")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.AddCookie(c)
	me := s.do(req)
	if me.Code != http.StatusOK || !bytes.Contains(me.Body.Bytes(), []byte(`"principal_id":"user-1"`)) {
		t.Fatalf("me: %d %s", me.Code, me.Body.String())
	}
	// No credentials → 401, never a fabricated identity.
	if rr := s.do(httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)); rr.Code != http.StatusUnauthorized {
		t.Fatalf("me without session: %d", rr.Code)
	}
}

func TestSession_CookieWriteRequiresOriginAndCSRF(t *testing.T) {
	s := newSessionHarness(t)
	c := sessionCookie(s.login(t, s.plain))
	sess, _ := s.store.Lookup(c.Value)

	mk := func(origin, csrf string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
		req.Host = "cyberos.test"
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if csrf != "" {
			req.Header.Set(auth.CSRFHeader, csrf)
		}
		req.AddCookie(c)
		return s.do(req)
	}
	if rr := mk("http://evil.test", sess.CSRFToken); rr.Code != http.StatusForbidden {
		t.Fatalf("cross-origin write: %d", rr.Code)
	}
	if rr := mk("", sess.CSRFToken); rr.Code != http.StatusForbidden {
		t.Fatalf("missing origin: %d", rr.Code)
	}
	if rr := mk("http://cyberos.test", ""); rr.Code != http.StatusForbidden {
		t.Fatalf("missing csrf: %d", rr.Code)
	}
	if rr := mk("http://cyberos.test", sess.CSRFToken); rr.Code != http.StatusNoContent {
		t.Fatalf("valid logout: %d", rr.Code)
	}
	// Session is dead after logout.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard", nil)
	req.AddCookie(c)
	if rr := s.do(req); rr.Code != http.StatusUnauthorized {
		t.Fatalf("session must be invalid after logout: %d", rr.Code)
	}
}

func TestSession_ExpiryAndSPAPathsPublic(t *testing.T) {
	st := auth.NewSessionStore(time.Minute, time.Minute)
	now := time.Now()
	st.Now = func() time.Time { return now }
	tok, _, err := st.Create(context.Background(), auth.Identity{})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	if _, ok := st.Lookup(tok); ok {
		t.Fatal("expired session must not resolve")
	}
	s := newSessionHarness(t)
	if rr := s.do(httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)); rr.Code != http.StatusUnauthorized {
		t.Fatalf("API must stay protected: %d", rr.Code)
	}
}

func TestSession_QueryTokenOnlyForSSE(t *testing.T) {
	s := newSessionHarness(t)
	rr := s.do(httptest.NewRequest(http.MethodGet, "/api/v1/dashboard?access_token="+s.plain, nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("query token must not authenticate non-SSE routes: %d", rr.Code)
	}
}

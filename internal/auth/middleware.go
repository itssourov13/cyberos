package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// SessionCookieName is the HttpOnly session cookie. The "__Host-" prefix is
// not used so plain-HTTP loopback development works; production sets Secure.
const SessionCookieName = "cyberos_session"

// CSRFHeader carries the per-session CSRF token on cookie-authenticated writes.
const CSRFHeader = "X-CSRF-Token"

type ctxSessionKey struct{}

// SessionTokenFromContext returns the raw session token for cookie-authenticated requests.
func SessionTokenFromContext(ctx context.Context) (string, bool) {
	t, ok := ctx.Value(ctxSessionKey{}).(string)
	return t, ok && t != ""
}

// Middleware authenticates requests and attaches Identity to the request context.
// It does not authorize execution.
type Middleware struct {
	Auth Authenticator
	// Sessions enables cookie sessions. Nil = header credentials only.
	Sessions *SessionStore
	// AllowedOrigins lists extra trusted origins (scheme://host[:port]) for
	// cookie-authenticated writes; same-origin is always allowed.
	AllowedOrigins []string
	// CookieSecure sets the Secure attribute on session cookies (always on in production).
	CookieSecure bool
	// Optional paths that skip authentication (health/version/login).
	PublicPaths map[string]bool
}

// NewMiddleware constructs auth middleware.
func NewMiddleware(a Authenticator) *Middleware {
	return &Middleware{
		Auth: a,
		PublicPaths: map[string]bool{
			"/api/v1/health":     true,
			"/api/v1/version":    true,
			"/api/v1/auth/login": true,
		},
	}
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}

func forbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// OriginAllowed reports whether the request Origin is same-origin or allow-listed.
func (m *Middleware) OriginAllowed(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		return false
	}
	u, err := url.Parse(o)
	if err != nil || u.Host == "" {
		return false
	}
	if u.Host == r.Host {
		return true
	}
	for _, a := range m.AllowedOrigins {
		if strings.EqualFold(strings.TrimRight(a, "/"), o) {
			return true
		}
	}
	return false
}

// Wrap returns an http.Handler that authenticates then calls next.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.Auth == nil {
			unauthorized(w, "auth not configured")
			return
		}
		// Only the API is protected; the SPA shell and static assets are public
		// and contain no data. Every data route lives under /api/.
		if !strings.HasPrefix(r.URL.Path, "/api/") || m.PublicPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		cred := r.Header.Get("Authorization")
		if cred == "" {
			cred = r.Header.Get("X-API-Key")
		}
		// A query token is accepted only for the SSE stream (EventSource cannot set headers).
		if cred == "" && r.URL.Path == "/api/v1/events/stream" {
			cred = r.URL.Query().Get("access_token")
		}
		if cred != "" {
			id, err := m.Auth.Authenticate(r.Context(), cred)
			if err != nil {
				unauthorized(w, "unauthorized")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), id)))
			return
		}
		if m.Sessions != nil {
			if c, err := r.Cookie(SessionCookieName); err == nil {
				sess, ok := m.Sessions.Lookup(c.Value)
				if !ok {
					unauthorized(w, "session expired")
					return
				}
				if !isSafeMethod(r.Method) {
					if !m.OriginAllowed(r) {
						forbidden(w, "cross-origin request rejected")
						return
					}
					if !CSRFValid(sess, r.Header.Get(CSRFHeader)) {
						forbidden(w, "csrf token invalid")
						return
					}
				}
				ctx := WithIdentity(r.Context(), sess.Identity)
				ctx = context.WithValue(ctx, ctxSessionKey{}, c.Value)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		unauthorized(w, "unauthorized")
	})
}

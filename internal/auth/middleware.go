package auth

import (
	"net/http"
	"strings"
)

// Middleware authenticates requests and attaches Identity to the request context.
// It does not authorize execution.
type Middleware struct {
	Auth Authenticator
	// Optional paths that skip authentication (health/version).
	PublicPaths map[string]bool
}

// NewMiddleware constructs auth middleware.
func NewMiddleware(a Authenticator) *Middleware {
	return &Middleware{
		Auth: a,
		PublicPaths: map[string]bool{
			"/api/v1/health":  true,
			"/api/v1/version": true,
		},
	}
}

// Wrap returns an http.Handler that authenticates then calls next.
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if m == nil || m.Auth == nil {
			http.Error(w, `{"error":"auth not configured"}`, http.StatusUnauthorized)
			return
		}
		if m.PublicPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}
		// UI static may be public for foundation demos
		if strings.HasPrefix(r.URL.Path, "/ui/") || r.URL.Path == "/" {
			next.ServeHTTP(w, r)
			return
		}
		cred := r.Header.Get("Authorization")
		if cred == "" {
			cred = r.Header.Get("X-API-Key")
		}
		if cred == "" {
			// SSE often uses query token
			cred = r.URL.Query().Get("access_token")
		}
		id, err := m.Auth.Authenticate(r.Context(), cred)
		if err != nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ctx := WithIdentity(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

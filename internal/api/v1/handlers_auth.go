package v1

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"cyberos/internal/auth"
)

var (
	loginLimiterOnce sync.Once
	loginLimiter     *fixedWindowLimiter
)

func loginLimit() *fixedWindowLimiter {
	loginLimiterOnce.Do(func() { loginLimiter = newFixedWindowLimiter(10, time.Minute) })
	return loginLimiter
}

type sessionView struct {
	AuthRequired  bool      `json:"auth_required"`
	Authenticated bool      `json:"authenticated"`
	PrincipalID   string    `json:"principal_id,omitempty"`
	TenantID      string    `json:"tenant_id,omitempty"`
	CSRFToken     string    `json:"csrf_token,omitempty"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
}

func (s *Server) cookieSecure(r *http.Request) bool {
	return s.Auth != nil && (s.Auth.CookieSecure || r.TLS != nil)
}

func (s *Server) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, exp time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: auth.SessionCookieName, Value: token, Path: "/",
		Expires: exp, HttpOnly: true, Secure: s.cookieSecure(r), SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name: auth.SessionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: s.cookieSecure(r), SameSite: http.SameSiteStrictMode,
	})
}

// handleLogin exchanges an API key for a server-side session. The credential
// is verified by the existing authenticator; the browser receives only an
// opaque HttpOnly cookie and a CSRF token, never the key back.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Auth == nil || s.Auth.Sessions == nil {
		writeErr(w, http.StatusServiceUnavailable, "authentication is not enabled on this server")
		return
	}
	// Login CSRF / cross-site form posts: require same-origin when Origin is sent.
	if o := r.Header.Get("Origin"); o != "" && !s.Auth.OriginAllowed(r) {
		writeErr(w, http.StatusForbidden, "cross-origin request rejected")
		return
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ok, retry := loginLimit().allow("login|" + host); !ok {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		writeErr(w, http.StatusTooManyRequests, "too many sign-in attempts")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var body struct {
		APIKey string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.APIKey) == "" {
		writeErr(w, http.StatusUnprocessableEntity, "api_key is required")
		return
	}
	id, err := s.Auth.Auth.Authenticate(r.Context(), body.APIKey)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	tok, sess, err := s.Auth.Sessions.Create(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "could not establish session")
		return
	}
	s.setSessionCookie(w, r, tok, sess.ExpiresAt)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, sessionView{
		AuthRequired: true, Authenticated: true,
		PrincipalID: string(id.Principal.ID), TenantID: string(id.TenantID),
		CSRFToken: sess.CSRFToken, ExpiresAt: sess.ExpiresAt,
	})
}

// handleMe reports the current session. With auth disabled (explicit dev mode)
// it says so instead of pretending a user exists.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if s.Auth == nil {
		writeJSON(w, http.StatusOK, sessionView{AuthRequired: false})
		return
	}
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		writeErr(w, http.StatusUnauthorized, "authentication required")
		return
	}
	view := sessionView{AuthRequired: true, Authenticated: true, PrincipalID: string(p.ID)}
	if t, ok := auth.TenantFromContext(r.Context()); ok {
		view.TenantID = string(t)
	}
	if tok, ok := auth.SessionTokenFromContext(r.Context()); ok && s.Auth.Sessions != nil {
		sess, found, err := s.Auth.Sessions.LookupContext(r.Context(), tok)
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, "session storage unavailable")
			return
		}
		if found {
			view.CSRFToken = sess.CSRFToken
			view.ExpiresAt = sess.ExpiresAt
		}
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.Auth != nil && s.Auth.Sessions != nil {
		if tok, ok := auth.SessionTokenFromContext(r.Context()); ok {
			if err := s.Auth.Sessions.DeleteContext(r.Context(), tok); err != nil {
				writeErr(w, http.StatusServiceUnavailable, "could not invalidate session")
				return
			}
		}
	}
	s.clearSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

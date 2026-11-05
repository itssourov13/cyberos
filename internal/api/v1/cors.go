package v1

import (
	"net/http"
	"strings"
)

// cors applies a strict credentialed CORS policy. No wildcard origin is ever
// emitted: every allowed origin must match the configured allow-list or the
// exact scheme/host of the request.
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}
		allowed := s.Auth != nil && s.Auth.OriginAllowed(r)
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			if !allowed {
				writeErr(w, http.StatusForbidden, "CORS origin not allowed")
				return
			}
			setCORSHeaders(w, origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-API-Key, X-CSRF-Token, X-CyberOS-Project-ID")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if allowed {
			setCORSHeaders(w, origin)
		}
		next.ServeHTTP(w, r)
	})
}

func setCORSHeaders(w http.ResponseWriter, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	appendVary(w.Header(), "Origin")
}

func appendVary(h http.Header, value string) {
	for _, existing := range h.Values("Vary") {
		for _, token := range strings.Split(existing, ",") {
			if strings.EqualFold(strings.TrimSpace(token), value) {
				return
			}
		}
	}
	h.Add("Vary", value)
}

// bodyLimit protects every write route, including endpoints that do not have
// their own domain-specific decoder limit.
func (s *Server) bodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			s.limitBody(w, r)
		}
		next.ServeHTTP(w, r)
	})
}

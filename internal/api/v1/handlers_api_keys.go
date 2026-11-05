package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

// apiKeySessionInvalidator eagerly removes browser sessions after revocation.
// PostgreSQL also enforces key/session invalidation transactionally.
type apiKeySessionInvalidator interface {
	DeleteByAPIKey(context.Context, string) error
}

func (s *Server) registerAPIKeyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/keys", s.handleAPIKeyIssue)
	mux.HandleFunc("POST /api/v1/auth/keys/{id}/revoke", s.handleAPIKeyRevoke)
}

func (s *Server) operatorAuthenticator(w http.ResponseWriter, r *http.Request) (*auth.APIKeyAuthenticator, bool) {
	if !s.requireOperator(w, r) {
		return nil, false
	}
	if s.Auth == nil {
		writeErr(w, http.StatusServiceUnavailable, "API-key administration requires configured authentication")
		return nil, false
	}
	a, ok := s.Auth.Auth.(*auth.APIKeyAuthenticator)
	if !ok || a == nil || a.Store == nil {
		writeErr(w, http.StatusNotImplemented, "API-key administration is not configured")
		return nil, false
	}
	return a, true
}

func (s *Server) handleAPIKeyIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	a, ok := s.operatorAuthenticator(w, r)
	if !ok {
		return
	}
	var body struct {
		PrincipalID string `json:"principal_id"`
		TenantID    string `json:"tenant_id"`
		Role        string `json:"role"`
		TTLSeconds  int64  `json:"ttl_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	role := auth.ParseRole(body.Role)
	if body.PrincipalID == "" || body.TenantID == "" || role == "" || role == auth.RoleOperator {
		writeErr(w, http.StatusBadRequest, "principal_id, tenant_id and a tenant role (viewer, analyst, admin) are required")
		return
	}
	const maxTTLSeconds = int64(365 * 24 * 60 * 60)
	if body.TTLSeconds < 3600 || body.TTLSeconds > maxTTLSeconds {
		writeErr(w, http.StatusBadRequest, "ttl_seconds must be between 3600 and 31536000")
		return
	}
	if s.SDK == nil || s.SDK.Identity == nil {
		writeErr(w, http.StatusServiceUnavailable, "identity service unavailable")
		return
	}
	if _, err := s.SDK.Identity.RequireActive(r.Context(), contracts.PrincipalID(body.PrincipalID)); err != nil {
		writeErr(w, http.StatusBadRequest, "principal must exist and be active")
		return
	}
	plain, key, err := a.IssueWithRole(
		r.Context(), contracts.PrincipalID(body.PrincipalID), contracts.TenantID(body.TenantID),
		role, time.Duration(body.TTLSeconds)*time.Second,
	)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	actor, _ := auth.PrincipalFromContext(r.Context())
	if s.SDK.Audit != nil {
		reason := "api key " + key.ID + " issued for tenant " + body.TenantID + " with role " + string(role)
		if _, err := s.SDK.Audit.RecordAdministrative("auth.api_key.issued", actor, "", reason); err != nil {
			_ = a.Store.Revoke(r.Context(), key.ID)
			if s.Auth.Sessions != nil {
				if invalidator, ok := s.Auth.Sessions.(apiKeySessionInvalidator); ok {
					_ = invalidator.DeleteByAPIKey(r.Context(), key.ID)
				}
			}
			writeErr(w, http.StatusServiceUnavailable, "key issuance was rolled back because audit persistence failed")
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": key.ID, "prefix": key.Prefix, "principal_id": key.PrincipalID,
		"tenant_id": key.TenantID, "role": key.Role, "expires_at": key.ExpiresAt,
		"api_key": plain, "shown_once": true,
	})
}

func (s *Server) handleAPIKeyRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	a, ok := s.operatorAuthenticator(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "key id required")
		return
	}
	if err := a.Store.Revoke(r.Context(), id); err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	if s.Auth.Sessions != nil {
		if invalidator, ok := s.Auth.Sessions.(apiKeySessionInvalidator); ok {
			if err := invalidator.DeleteByAPIKey(r.Context(), id); err != nil {
				writeErr(w, http.StatusServiceUnavailable, "key revoked but session cleanup failed")
				return
			}
		}
	}
	actor, _ := auth.PrincipalFromContext(r.Context())
	if s.SDK != nil && s.SDK.Audit != nil {
		if _, err := s.SDK.Audit.RecordAdministrative("auth.api_key.revoked", actor, "", "api key "+id+" revoked"); err != nil {
			writeErr(w, http.StatusServiceUnavailable, "key revoked but audit persistence failed")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

package v1

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

// registerGrantRevocationRoutes exposes durable, attributable revocation. A
// revoke only takes effect while the ledger state is ISSUED; CLAIMED grants are
// intentionally non-revocable because the execution outcome is uncertain.
func (s *Server) registerGrantRevocationRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/grants/{id}/revoke", s.handleGrantRevoke)
}

func (s *Server) handleGrantRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.Broker == nil || s.SDK.Broker.Ledger == nil {
		writeErr(w, http.StatusServiceUnavailable, "durable grant ledger unavailable")
		return
	}
	actor, authenticated := auth.PrincipalFromContext(r.Context())
	if s.Auth != nil && !authenticated {
		writeErr(w, http.StatusUnauthorized, "authentication required")
		return
	}
	operator := callerTenant(r) == ""
	if !operator {
		if !s.requirePermission(w, r, auth.PermExecute) {
			return
		}
	}
	id := contracts.ExecutionGrantID(strings.TrimSpace(r.PathValue("id")))
	if id == "" {
		writeErr(w, http.StatusBadRequest, "grant id required")
		return
	}
	record, err := s.SDK.Broker.Ledger.Get(r.Context(), id)
	if errors.Is(err, contracts.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "grant not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "grant ledger unavailable")
		return
	}
	if !operator {
		if record.ProjectID == "" || !enforceProject(w, r, record.ProjectID) {
			return
		}
		if record.PrincipalID != actor.ID && auth.RoleFromContext(r.Context()) != auth.RoleAdmin {
			writeErr(w, http.StatusForbidden, "only the grant owner or a project administrator may revoke this grant")
			return
		}
	}
	// Check terminal state only after resource authorization to avoid leaking
	// another tenant's grant state through an idempotent response.
	if string(record.State) == "REVOKED" {
		writeJSON(w, http.StatusOK, map[string]any{
			"grant_id": id, "state": "REVOKED", "already_revoked": true,
		})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		writeErr(w, http.StatusBadRequest, "reason is required")
		return
	}
	actorRef := actor
	if !authenticated {
		actorRef = contracts.PrincipalRef{ID: "dev-operator", Type: contracts.PrincipalTypeService}
	}
	if err := s.SDK.Broker.RevokeGrantDurable(r.Context(), id, actorRef.ID, reason); err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	if s.SDK.Audit != nil {
		if _, err := s.SDK.Audit.RecordGrantRevoked(id, record.PrincipalID, record.ProjectID, record.CorrelationID, actorRef, reason); err != nil {
			writeErr(w, http.StatusServiceUnavailable, "grant revoked but durable audit write failed")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"grant_id": id, "state": "REVOKED", "revoked_by": actorRef.ID,
	})
}

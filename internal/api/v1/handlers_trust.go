package v1

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
	"cyberos/internal/tools"
)

func (s *Server) registerTrustRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/trust/packs", s.handleTrustPackGet)
	mux.HandleFunc("POST /api/v1/trust/packs/trust", s.handleTrustPackTrust)
	mux.HandleFunc("POST /api/v1/trust/packs/revoke", s.handleTrustPackRevoke)
	mux.HandleFunc("POST /api/v1/trust/signers", s.handleSignerRegister)
	mux.HandleFunc("GET /api/v1/trust/signers/{id}", s.handleSignerGet)
	mux.HandleFunc("POST /api/v1/trust/signers/{id}/disable", s.handleSignerDisable)
	mux.HandleFunc("POST /api/v1/trust/signers/{id}/revoke", s.handleSignerRevoke)
}

func (s *Server) trustOperator(w http.ResponseWriter, r *http.Request) (contracts.PrincipalRef, bool) {
	if !s.requireOperator(w, r) {
		return contracts.PrincipalRef{}, false
	}
	if s.TrustAdmin == nil {
		writeErr(w, http.StatusServiceUnavailable, "trust administration is not configured")
		return contracts.PrincipalRef{}, false
	}
	actor, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		if s.Auth != nil {
			writeErr(w, http.StatusUnauthorized, "operator identity required")
			return contracts.PrincipalRef{}, false
		}
		actor = contracts.PrincipalRef{ID: "dev-operator", Type: contracts.PrincipalTypeService}
	}
	return actor, true
}

func (s *Server) requireTrustProject(w http.ResponseWriter, r *http.Request, project contracts.ProjectID) bool {
	if project == "" {
		writeErr(w, http.StatusBadRequest, "project_id required")
		return false
	}
	tenant, exists := projectOwner(r, project)
	if !exists {
		writeErr(w, http.StatusNotFound, "project not found")
		return false
	}
	// Platform operators are tenant-less by design; project ownership still
	// must resolve, but there is no caller tenant to compare with the resource.
	if callerTenant(r) == "" {
		return enforceProject(w, r, project)
	}
	return enforceResource(w, r, tenant, project)
}

func (s *Server) handleTrustPackGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := s.trustOperator(w, r); !ok {
		return
	}
	project := contracts.ProjectID(strings.TrimSpace(r.URL.Query().Get("project_id")))
	digest := strings.TrimSpace(r.URL.Query().Get("digest"))
	if !s.requireTrustProject(w, r, project) {
		return
	}
	if err := tools.ValidateDigestFormat(digest); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid pack digest")
		return
	}
	store := s.TrustAdmin.Store
	var trusted, revoked bool
	var err error
	if reader, ok := store.(interface {
		IsTrustedCtx(context.Context, contracts.ProjectID, string) (bool, error)
		IsRevokedCtx(context.Context, contracts.ProjectID, string) (bool, error)
	}); ok {
		trusted, err = reader.IsTrustedCtx(r.Context(), project, digest)
		if err == nil {
			revoked, err = reader.IsRevokedCtx(r.Context(), project, digest)
		}
	} else {
		trusted = store.IsTrusted(project, digest)
		revoked = store.IsRevoked(project, digest)
	}
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "trust store unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project_id": project, "digest": digest, "trusted": trusted, "revoked": revoked,
	})
}

func (s *Server) handleTrustPackTrust(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	actor, ok := s.trustOperator(w, r)
	if !ok {
		return
	}
	var body struct {
		ProjectID     string `json:"project_id"`
		Digest        string `json:"digest"`
		Reason        string `json:"reason"`
		CorrelationID string `json:"correlation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	project := contracts.ProjectID(strings.TrimSpace(body.ProjectID))
	digest := strings.TrimSpace(body.Digest)
	if !s.requireTrustProject(w, r, project) {
		return
	}
	if err := tools.ValidateDigestFormat(digest); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid pack digest")
		return
	}
	if err := s.TrustAdmin.TrustPack(actor, project, digest, body.CorrelationID, body.Reason); err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project_id": project, "digest": digest, "state": "TRUSTED"})
}

func (s *Server) handleTrustPackRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	actor, ok := s.trustOperator(w, r)
	if !ok {
		return
	}
	var body struct {
		ProjectID     string `json:"project_id"`
		Digest        string `json:"digest"`
		Reason        string `json:"reason"`
		CorrelationID string `json:"correlation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	project := contracts.ProjectID(strings.TrimSpace(body.ProjectID))
	digest := strings.TrimSpace(body.Digest)
	reason := strings.TrimSpace(body.Reason)
	if !s.requireTrustProject(w, r, project) {
		return
	}
	if reason == "" {
		writeErr(w, http.StatusBadRequest, "reason is required for pack revocation")
		return
	}
	if err := tools.ValidateDigestFormat(digest); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid pack digest")
		return
	}
	if err := s.TrustAdmin.RevokePack(actor, project, digest, reason, body.CorrelationID); err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project_id": project, "digest": digest, "state": "REVOKED"})
}

func (s *Server) handleSignerRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	actor, ok := s.trustOperator(w, r)
	if !ok {
		return
	}
	var body struct {
		KeyID         string `json:"key_id"`
		Algorithm     string `json:"algorithm"`
		PublicKey     string `json:"public_key"`
		CorrelationID string `json:"correlation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Algorithm == "" {
		body.Algorithm = tools.PackSigAlgorithm
	}
	if body.Algorithm != tools.PackSigAlgorithm || strings.TrimSpace(body.KeyID) == "" {
		writeErr(w, http.StatusBadRequest, "key_id and ed25519 algorithm are required")
		return
	}
	publicKey, err := base64.StdEncoding.DecodeString(body.PublicKey)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		writeErr(w, http.StatusBadRequest, "public_key must be base64-encoded Ed25519 public key material")
		return
	}
	key := tools.SignerKey{
		KeyID: strings.TrimSpace(body.KeyID), Algorithm: body.Algorithm,
		PublicKey: ed25519.PublicKey(publicKey), Status: tools.SignerKeyActive,
	}
	if err := s.TrustAdmin.RegisterSigner(actor, key, body.CorrelationID); err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusCreated, signerView(key))
}

func (s *Server) handleSignerGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, ok := s.trustOperator(w, r); !ok {
		return
	}
	key, err := s.TrustAdmin.GetSigner(r.PathValue("id"))
	if errors.Is(err, contracts.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "signer not found")
		return
	}
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusOK, signerView(key))
}

func (s *Server) handleSignerDisable(w http.ResponseWriter, r *http.Request) {
	s.signerTransition(w, r, "disable")
}

func (s *Server) handleSignerRevoke(w http.ResponseWriter, r *http.Request) {
	s.signerTransition(w, r, "revoke")
}

func (s *Server) signerTransition(w http.ResponseWriter, r *http.Request, action string) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	actor, ok := s.trustOperator(w, r)
	if !ok {
		return
	}
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeErr(w, http.StatusBadRequest, "signer id required")
		return
	}
	var err error
	correlation := "api-signer-" + id + "-" + action
	switch action {
	case "disable":
		err = s.TrustAdmin.DisableSigner(actor, id, correlation)
	case "revoke":
		err = s.TrustAdmin.RevokeSigner(actor, id, correlation)
	}
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	key, err := s.TrustAdmin.GetSigner(id)
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "signer status could not be reloaded")
		return
	}
	writeJSON(w, http.StatusOK, signerView(key))
}

func signerView(key tools.SignerKey) map[string]any {
	return map[string]any{
		"key_id": key.KeyID, "algorithm": key.Algorithm,
		"public_key": base64.StdEncoding.EncodeToString(key.PublicKey),
		"status":     key.Status, "created_at": key.CreatedAt,
	}
}

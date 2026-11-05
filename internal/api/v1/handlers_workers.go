package v1

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
	"cyberos/internal/execution"
	"cyberos/internal/execution/mtls"
)

// RegisterWorkers attaches worker management routes (identity only; no execution).
func (s *Server) registerWorkerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/projects/{project}/workers", s.handleWorkerRegister)
	mux.HandleFunc("GET /api/v1/projects/{project}/workers", s.handleWorkerList)
	mux.HandleFunc("GET /api/v1/projects/{project}/workers/{id}", s.handleWorkerGet)
	mux.HandleFunc("POST /api/v1/projects/{project}/workers/{id}/activate", s.handleWorkerActivate)
	mux.HandleFunc("POST /api/v1/projects/{project}/workers/{id}/suspend", s.handleWorkerSuspend)
	mux.HandleFunc("POST /api/v1/projects/{project}/workers/{id}/revoke", s.handleWorkerRevoke)
	mux.HandleFunc("POST /api/v1/projects/{project}/workers/{id}/heartbeat", s.handleWorkerHeartbeat)
}

func (s *Server) handleWorkerRegister(w http.ResponseWriter, r *http.Request) {
	if s.Workers == nil {
		writeErr(w, http.StatusNotImplemented, "workers not configured")
		return
	}
	s.limitBody(w, r)
	project := contracts.ProjectID(r.PathValue("project"))
	if !enforceProject(w, r, project) {
		return
	}
	if !s.requirePermission(w, r, auth.PermWorkerAdmin) {
		return
	}
	var body struct {
		ID              string            `json:"worker_id"`
		HostID          string            `json:"host_id"`
		TenantID        string            `json:"tenant_id"`
		CredentialRef   string            `json:"credential_ref"`
		CertFingerprint string            `json:"cert_fingerprint"`
		Capabilities    map[string]string `json:"capabilities"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	// Tenant ownership comes from the canonical project record, never the body.
	tenant, known := projectOwner(r, project)
	if !known {
		writeErr(w, http.StatusBadRequest, "unknown project")
		return
	}
	if body.TenantID != "" && contracts.TenantID(body.TenantID) != tenant {
		writeErr(w, http.StatusForbidden, "tenant_id does not match the project's tenant")
		return
	}
	fingerprint := strings.ToLower(strings.TrimSpace(body.CertFingerprint))
	if len(fingerprint) != 64 {
		writeErr(w, http.StatusBadRequest, "a SHA-256 worker certificate fingerprint is required")
		return
	}
	if _, err := hex.DecodeString(fingerprint); err != nil {
		writeErr(w, http.StatusBadRequest, "cert_fingerprint must be 64 hexadecimal characters")
		return
	}
	worker, err := s.Workers.Register(r.Context(), execution.Worker{
		ID: body.ID, ProjectID: project, TenantID: tenant,
		HostID: execution.HostID(body.HostID), CredentialRef: body.CredentialRef,
		CertFingerprint: fingerprint, Capabilities: body.Capabilities,
		Status: execution.WorkerPending,
	})
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	if s.SDK != nil && s.SDK.Audit != nil {
		actor, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			actor = contracts.PrincipalRef{ID: "dev", Type: contracts.PrincipalTypeHuman}
		}
		if _, auditErr := s.SDK.Audit.RecordAdministrative("worker.registered", actor, project, "worker "+worker.ID+" registered pending certificate activation"); auditErr != nil {
			_, _ = s.Workers.Revoke(r.Context(), project, worker.ID)
			writeErr(w, http.StatusServiceUnavailable, "worker registration was rolled back because audit persistence failed")
			return
		}
	}
	writeJSON(w, http.StatusCreated, workerView(worker))
}

func (s *Server) handleWorkerList(w http.ResponseWriter, r *http.Request) {
	if s.Workers == nil {
		writeErr(w, http.StatusNotImplemented, "workers not configured")
		return
	}
	project := contracts.ProjectID(r.PathValue("project"))
	if !enforceProject(w, r, project) {
		return
	}
	list, err := s.Workers.List(r.Context(), project)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	ct := callerTenant(r)
	views := make([]map[string]any, 0, len(list))
	for _, wk := range list {
		if ct != "" && wk.TenantID != ct {
			continue
		}
		views = append(views, workerView(wk))
	}
	writeJSON(w, http.StatusOK, map[string]any{"project_id": project, "workers": views})
}

func (s *Server) handleWorkerGet(w http.ResponseWriter, r *http.Request) {
	if s.Workers == nil {
		writeErr(w, http.StatusNotImplemented, "workers not configured")
		return
	}
	project := contracts.ProjectID(r.PathValue("project"))
	if !enforceProject(w, r, project) {
		return
	}
	worker, err := s.Workers.Get(r.Context(), project, r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, workerView(worker))
}

func (s *Server) handleWorkerActivate(w http.ResponseWriter, r *http.Request) {
	s.workerTransition(w, r, "activate")
}

func (s *Server) handleWorkerSuspend(w http.ResponseWriter, r *http.Request) {
	s.workerTransition(w, r, "suspend")
}
func (s *Server) handleWorkerRevoke(w http.ResponseWriter, r *http.Request) {
	s.workerTransition(w, r, "revoke")
}

func (s *Server) workerTransition(w http.ResponseWriter, r *http.Request, action string) {
	if s.Workers == nil {
		writeErr(w, http.StatusNotImplemented, "workers not configured")
		return
	}
	project := contracts.ProjectID(r.PathValue("project"))
	if !enforceProject(w, r, project) {
		return
	}
	if !s.requirePermission(w, r, auth.PermWorkerAdmin) {
		return
	}
	id := r.PathValue("id")
	var worker execution.Worker
	var err error
	switch action {
	case "activate":
		worker, err = s.Workers.Activate(r.Context(), project, id)
	case "suspend":
		worker, err = s.Workers.Suspend(r.Context(), project, id)
	case "revoke":
		worker, err = s.Workers.Revoke(r.Context(), project, id)
	default:
		writeErr(w, http.StatusBadRequest, "unknown action")
		return
	}
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	if s.SDK != nil && s.SDK.Audit != nil {
		actor, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			actor = contracts.PrincipalRef{ID: "dev", Type: contracts.PrincipalTypeHuman}
		}
		if _, auditErr := s.SDK.Audit.RecordAdministrative("worker."+action, actor, project, "worker "+id+" transitioned to "+string(worker.Status)); auditErr != nil {
			writeErr(w, http.StatusServiceUnavailable, "worker status changed but audit persistence failed")
			return
		}
	}
	writeJSON(w, http.StatusOK, workerView(worker))
}

func (s *Server) handleWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	if s.Workers == nil {
		writeErr(w, http.StatusNotImplemented, "workers not configured")
		return
	}
	project := contracts.ProjectID(r.PathValue("project"))
	if !enforceProject(w, r, project) {
		return
	}
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 {
		writeErr(w, http.StatusUnauthorized, "verified worker client certificate required")
		return
	}
	peer, err := mtls.ExtractPeerIdentity(*r.TLS)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "worker certificate invalid")
		return
	}
	id := r.PathValue("id")
	registered, err := s.Workers.Get(r.Context(), project, id)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "worker identity not registered")
		return
	}
	if _, err := s.Workers.AuthenticateTLS(r.Context(), project, id, string(registered.HostID), peer.Fingerprint); err != nil {
		writeErr(w, http.StatusForbidden, "worker identity rejected")
		return
	}
	worker, err := s.Workers.Heartbeat(r.Context(), project, id)
	if err != nil {
		writeErr(w, http.StatusForbidden, "heartbeat rejected")
		return
	}
	writeJSON(w, http.StatusOK, workerView(worker))
}

func workerView(w execution.Worker) map[string]any {
	return map[string]any{
		"worker_id": w.ID, "project_id": w.ProjectID, "host_id": w.HostID, "status": w.Status,
		"credential_ref": w.CredentialRef, "cert_fingerprint": w.CertFingerprint,
		"last_seen_at": w.LastSeenAt, "expires_at": w.ExpiresAt,
		// never expose private keys
	}
}

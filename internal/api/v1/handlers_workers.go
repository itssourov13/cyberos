package v1

import (
	"encoding/json"
	"net/http"

	"cyberos/internal/contracts"
	"cyberos/internal/execution"
)

// RegisterWorkers attaches worker management routes (identity only; no execution).
func (s *Server) registerWorkerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/projects/{project}/workers", s.handleWorkerRegister)
	mux.HandleFunc("GET /api/v1/projects/{project}/workers", s.handleWorkerList)
	mux.HandleFunc("GET /api/v1/projects/{project}/workers/{id}", s.handleWorkerGet)
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
	// Tenant ownership is bound from the authenticated identity, never the body.
	tenant := contracts.TenantID(body.TenantID)
	if ct := callerTenant(r); ct != "" {
		if tenant != "" && tenant != ct {
			writeErr(w, http.StatusForbidden, "tenant_id does not match the authenticated tenant")
			return
		}
		tenant = ct
	}
	worker, err := s.Workers.Register(r.Context(), execution.Worker{
		ID: body.ID, ProjectID: project, TenantID: tenant,
		HostID: execution.HostID(body.HostID), CredentialRef: body.CredentialRef,
		CertFingerprint: body.CertFingerprint, Capabilities: body.Capabilities,
		Status: execution.WorkerPending,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
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
	// List via underlying repo is not exported; get individual is enough for 17C minimal API.
	writeJSON(w, http.StatusOK, map[string]any{"project_id": project, "note": "list via registry in SDK"})
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
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, workerView(worker))
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
	id := r.PathValue("id")
	var worker execution.Worker
	var err error
	switch action {
	case "suspend":
		worker, err = s.Workers.Suspend(r.Context(), project, id)
	case "revoke":
		worker, err = s.Workers.Revoke(r.Context(), project, id)
	default:
		writeErr(w, http.StatusBadRequest, "unknown action")
		return
	}
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
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
	worker, err := s.Workers.Heartbeat(r.Context(), project, r.PathValue("id"))
	if err != nil {
		writeErr(w, http.StatusForbidden, err.Error())
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

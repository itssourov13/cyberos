package v1

import (
	"encoding/json"
	"net/http"

	"cyberos/internal/ai/autonomy"
	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

// handleAutonomyRuns routes POST /api/v1/autonomy/runs and GET list is not required for 14C.
func (s *Server) handleAutonomyRuns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.StartAutonomyRun(w, r)
	default:
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleAutonomyRunByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.GetAutonomyRun(w, r)
}

func (s *Server) handleAutonomyCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.CancelAutonomyRun(w, r)
}

func (s *Server) handleAutonomyResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.ResumeAutonomyRun(w, r)
}

func (s *Server) handleAutonomyAdvance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.AdvanceAutonomyRun(w, r)
}

// StartAutonomyRun POST /api/v1/autonomy/runs
func (s *Server) StartAutonomyRun(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.Autonomy == nil {
		writeErr(w, http.StatusNotImplemented, "autonomy not configured")
		return
	}
	s.limitBody(w, r)
	if !s.limitRequest(w, r, "autonomy") {
		return
	}
	if !s.requirePermission(w, r, auth.PermAutonomyControl) {
		return
	}
	var body autonomy.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.ProjectID == "" {
		writeErr(w, http.StatusBadRequest, "project_id required")
		return
	}
	if !enforceProject(w, r, body.ProjectID) {
		return
	}
	// Identity and ownership are server-authoritative: never trust client-supplied
	// principal, tenant or policy ownership fields.
	principal, ok := s.bindPrincipal(w, r, body.PrincipalID)
	if !ok {
		return
	}
	if principal == "" {
		principal = s.actorID(r)
	}
	tenant, known := projectOwner(r, body.ProjectID)
	if !known {
		writeErr(w, http.StatusBadRequest, "unknown project")
		return
	}
	if !s.scopeBelongsToProject(body.ProjectID, body.ScopeID) {
		writeErr(w, http.StatusBadRequest, "scope does not belong to project")
		return
	}
	body.PrincipalID = principal
	body.TenantID = tenant
	body.Policy.ProjectID = body.ProjectID
	body.Policy.TenantID = tenant
	run, err := s.SDK.StartAutonomy(r.Context(), body)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusCreated, autonomyView(run))
}

// GetAutonomyRun GET /api/v1/autonomy/runs/{id}
func (s *Server) GetAutonomyRun(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.Autonomy == nil {
		writeErr(w, http.StatusNotImplemented, "autonomy not configured")
		return
	}
	id := r.PathValue("id")
	run, err := s.SDK.GetAutonomy(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if !enforceObject(w, r, run.TenantID, run.ProjectID) {
		return
	}
	writeJSON(w, http.StatusOK, autonomyView(run))
}

// CancelAutonomyRun POST /api/v1/autonomy/runs/{id}/cancel
func (s *Server) CancelAutonomyRun(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.Autonomy == nil {
		writeErr(w, http.StatusNotImplemented, "autonomy not configured")
		return
	}
	s.limitBody(w, r)
	if !s.limitRequest(w, r, "autonomy") {
		return
	}
	id := r.PathValue("id")
	run, err := s.SDK.GetAutonomy(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if !enforceObject(w, r, run.TenantID, run.ProjectID) {
		return
	}
	if !s.requirePermission(w, r, auth.PermAutonomyControl) {
		return
	}
	run, err = s.SDK.CancelAutonomy(id)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, autonomyView(run))
}

// ResumeAutonomyRun POST /api/v1/autonomy/runs/{id}/resume
func (s *Server) ResumeAutonomyRun(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.Autonomy == nil {
		writeErr(w, http.StatusNotImplemented, "autonomy not configured")
		return
	}
	s.limitBody(w, r)
	if !s.limitRequest(w, r, "autonomy") {
		return
	}
	if !s.requirePermission(w, r, auth.PermAutonomyControl) {
		return
	}
	id := r.PathValue("id")
	var body struct {
		ProjectID contracts.ProjectID `json:"project_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.ProjectID == "" {
		writeErr(w, http.StatusBadRequest, "project_id required")
		return
	}
	// The project is only a lookup key: it must be owned by the caller, and the
	// loaded run must actually belong to it.
	if !enforceProject(w, r, body.ProjectID) {
		return
	}
	run, err := s.SDK.ResumeAutonomy(r.Context(), body.ProjectID, id)
	if err != nil || run == nil || run.ProjectID != body.ProjectID {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if !enforceObject(w, r, run.TenantID, run.ProjectID) {
		return
	}
	writeJSON(w, http.StatusOK, autonomyView(run))
}

// AdvanceAutonomyRun POST /api/v1/autonomy/runs/{id}/advance
func (s *Server) AdvanceAutonomyRun(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.AutonomyHandoff == nil {
		writeErr(w, http.StatusNotImplemented, "autonomy handoff not configured")
		return
	}
	s.limitBody(w, r)
	if !s.limitRequest(w, r, "autonomy") {
		return
	}
	if !s.requirePermission(w, r, auth.PermAutonomyControl) {
		return
	}
	id := r.PathValue("id")
	var body struct {
		ProjectID     contracts.ProjectID `json:"project_id"`
		WorkflowRunID string              `json:"workflow_run_id"`
		// tenant_id / scope_id in a payload are ignored: both come from the stored run.
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	run, err := s.SDK.GetAutonomy(id)
	if err != nil && body.ProjectID != "" && s.SDK.Autonomy != nil {
		// Not loaded in this instance: load it under a project the caller owns.
		if !enforceProject(w, r, body.ProjectID) {
			return
		}
		run, err = s.SDK.ResumeAutonomy(r.Context(), body.ProjectID, id)
	}
	if err != nil || run == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if !enforceObject(w, r, run.TenantID, run.ProjectID) {
		return
	}
	if body.ProjectID != "" && body.ProjectID != run.ProjectID {
		writeErr(w, http.StatusForbidden, "cross-project access denied")
		return
	}
	res, err := s.SDK.AdvanceAutonomy(r.Context(), autonomy.HandoffRequest{
		RunID: id, ProjectID: run.ProjectID, TenantID: run.TenantID,
		ScopeID: run.ScopeID, WorkflowRunID: body.WorkflowRunID,
	})
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"run":                 autonomyView(res.Run),
		"policy_decision":     res.PolicyDecision,
		"orchestrator_called": res.OrchestratorCalled,
		"may_advance":         res.MayAdvance,
	})
}

func autonomyView(run *autonomy.AutonomyRun) map[string]any {
	if run == nil {
		return nil
	}
	return map[string]any{
		"run_id":          run.ID,
		"state":           run.State,
		"mode":            run.Policy.Mode,
		"project_id":      run.ProjectID,
		"tenant_id":       run.TenantID,
		"scope_id":        run.ScopeID,
		"goal":            run.Goal,
		"steps_consumed":  run.StepsConsumed,
		"iteration":       run.Iteration,
		"plan_id":         run.CurrentPlanID,
		"analysis_id":     run.AnalysisID,
		"workflow_def_id": run.WorkflowDefID,
		"stop_reason":     run.StopReason,
		"approval_state":  run.ApprovalState,
		"started_at":      run.StartedAt,
		"updated_at":      run.UpdatedAt,
	}
}

func (s *Server) handleAutonomyApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.ApproveAutonomyRun(w, r)
}

func (s *Server) handleAutonomyReject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	s.RejectAutonomyRun(w, r)
}

// ApproveAutonomyRun POST /api/v1/autonomy/runs/{id}/approve — ASSISTED control only (not a Grant).
func (s *Server) ApproveAutonomyRun(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.Autonomy == nil {
		writeErr(w, http.StatusNotImplemented, "autonomy not configured")
		return
	}
	s.limitBody(w, r)
	if !s.limitRequest(w, r, "autonomy") {
		return
	}
	id := r.PathValue("id")
	run, err := s.SDK.GetAutonomy(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if !enforceObject(w, r, run.TenantID, run.ProjectID) {
		return
	}
	if !s.requirePermission(w, r, auth.PermAutonomyApprove) {
		return
	}
	run, err = s.SDK.Autonomy.Approve(id)
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, autonomyView(run))
}

// RejectAutonomyRun POST /api/v1/autonomy/runs/{id}/reject — ASSISTED control only.
func (s *Server) RejectAutonomyRun(w http.ResponseWriter, r *http.Request) {
	if s.SDK == nil || s.SDK.Autonomy == nil {
		writeErr(w, http.StatusNotImplemented, "autonomy not configured")
		return
	}
	s.limitBody(w, r)
	if !s.limitRequest(w, r, "autonomy") {
		return
	}
	id := r.PathValue("id")
	run, err := s.SDK.GetAutonomy(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if !enforceObject(w, r, run.TenantID, run.ProjectID) {
		return
	}
	if !s.requirePermission(w, r, auth.PermAutonomyApprove) {
		return
	}
	run, err = s.SDK.Autonomy.Reject(id)
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, autonomyView(run))
}

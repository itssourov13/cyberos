package v1

import (
	"encoding/json"
	"net/http"

	"cyberos/internal/ai/autonomy"
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
	var body autonomy.StartRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !enforceProject(w, r, body.ProjectID) {
		return
	}
	run, err := s.SDK.StartAutonomy(r.Context(), body)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
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
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if !enforceProject(w, r, run.ProjectID) {
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
	id := r.PathValue("id")
	run, err := s.SDK.GetAutonomy(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if !enforceProject(w, r, run.ProjectID) {
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
	id := r.PathValue("id")
	var body struct {
		ProjectID contracts.ProjectID `json:"project_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.ProjectID == "" {
		writeErr(w, http.StatusBadRequest, "project_id required")
		return
	}
	if !enforceProject(w, r, body.ProjectID) {
		return
	}
	run, err := s.SDK.ResumeAutonomy(r.Context(), body.ProjectID, id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
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
	id := r.PathValue("id")
	var body struct {
		ProjectID     contracts.ProjectID `json:"project_id"`
		TenantID      contracts.TenantID  `json:"tenant_id"`
		ScopeID       contracts.ScopeID   `json:"scope_id"`
		WorkflowRunID string              `json:"workflow_run_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.ProjectID == "" {
		writeErr(w, http.StatusBadRequest, "project_id required")
		return
	}
	if !enforceProject(w, r, body.ProjectID) {
		return
	}
	res, err := s.SDK.AdvanceAutonomy(r.Context(), autonomy.HandoffRequest{
		RunID: id, ProjectID: body.ProjectID, TenantID: body.TenantID,
		ScopeID: body.ScopeID, WorkflowRunID: body.WorkflowRunID,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
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
		"started_at":      run.StartedAt,
		"updated_at":      run.UpdatedAt,
	}
}

package v1

import (
	"net/http"

	"cyberos/internal/contracts"
)

func (s *Server) handleFindings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.Findings == nil {
		writeJSON(w, http.StatusOK, map[string]any{"findings": []any{}})
		return
	}
	project := r.URL.Query().Get("project_id")
	assessment := r.URL.Query().Get("assessment_id")
	if assessment != "" {
		writeJSON(w, http.StatusOK, map[string]any{"findings": s.SDK.Findings.ListByAssessment(assessment)})
		return
	}
	if project != "" {
		writeJSON(w, http.StatusOK, map[string]any{"findings": s.SDK.Findings.ListByProject(contracts.ProjectID(project))})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"findings": s.SDK.Findings.ListByProject("")})
}

func (s *Server) handleFindingByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.Findings == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	f, err := s.SDK.Findings.Get(contracts.FindingID(id))
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) handleAssessmentFindings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.Findings == nil {
		writeJSON(w, http.StatusOK, map[string]any{"findings": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"findings": s.SDK.Findings.ListByAssessment(id)})
}

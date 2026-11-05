package v1

import (
	"net/http"

	"cyberos/internal/contracts"
	domainfinding "cyberos/internal/domain/finding"
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
	assessment := r.URL.Query().Get("assessment_id")
	if assessment != "" {
		a, ok := s.loadAssessment(w, r, assessment)
		if !ok {
			return
		}
		list := s.SDK.Findings.ListByAssessment(assessment)
		writeJSON(w, http.StatusOK, map[string]any{"findings": filterFindingsByProject(list, a.ProjectID)})
		return
	}
	project, ok := requireProjectScope(w, r)
	if !ok {
		return
	}
	if project == "" {
		writeJSON(w, http.StatusOK, map[string]any{"findings": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"findings": s.SDK.Findings.ListByProject(project)})
}

func filterFindingsByProject(list []*domainfinding.Finding, proj contracts.ProjectID) []*domainfinding.Finding {
	if proj == "" {
		return list
	}
	out := make([]*domainfinding.Finding, 0, len(list))
	for _, f := range list {
		if f.ProjectID == proj {
			out = append(out, f)
		}
	}
	return out
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
	if !enforceObject(w, r, "", f.ProjectID) {
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
	a, ok := s.loadAssessment(w, r, id)
	if !ok {
		return
	}
	if s.SDK.Findings == nil {
		writeJSON(w, http.StatusOK, map[string]any{"findings": []any{}})
		return
	}
	list := s.SDK.Findings.ListByAssessment(id)
	writeJSON(w, http.StatusOK, map[string]any{"findings": filterFindingsByProject(list, a.ProjectID)})
}

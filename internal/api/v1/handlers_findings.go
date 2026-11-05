package v1

import (
	"errors"
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
		list, err := s.SDK.Findings.ListByAssessmentPage(r.Context(), assessment, 500, 0)
		if err != nil {
			writeErr(w, statusForErr(err), publicErr(err))
			return
		}
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
	list, err := s.SDK.Findings.ListByProjectPage(r.Context(), project, 500, 0)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"findings": list})
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
	if errors.Is(err, contracts.ErrNotFound) || (err == nil && f == nil) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
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
	list, err := s.SDK.Findings.ListByAssessmentPage(r.Context(), id, 500, 0)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"findings": filterFindingsByProject(list, a.ProjectID)})
}

package v1

import (
	"net/http"

	"cyberos/internal/contracts"
)

func (s *Server) handleRisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.RiskPrio == nil {
		writeJSON(w, http.StatusOK, map[string]any{"risks": []any{}})
		return
	}
	assessment := r.URL.Query().Get("assessment_id")
	project, ok := requireProjectScope(w, r)
	if !ok {
		return
	}
	if assessment != "" {
		list := s.SDK.RiskPrio.ListByAssessment(assessment)
		if project != "" {
			filtered := list[:0]
			for _, ev := range list {
				if ev.ProjectID == project {
					filtered = append(filtered, ev)
				}
			}
			list = filtered
		}
		writeJSON(w, http.StatusOK, map[string]any{"risks": list})
		return
	}
	if project == "" {
		writeJSON(w, http.StatusOK, map[string]any{"risks": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"risks": s.SDK.RiskPrio.ListByProject(project)})
}

func (s *Server) handleRiskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.RiskPrio == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	ev, err := s.SDK.RiskPrio.Get(id)
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if !enforceResource(w, r, "", ev.ProjectID) {
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (s *Server) handleFindingRisk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.RiskPrio == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	ev, err := s.SDK.RiskPrio.GetByFinding(contracts.FindingID(id))
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	if !enforceResource(w, r, "", ev.ProjectID) {
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

func (s *Server) handleAssessmentRiskSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.RiskPrio == nil {
		writeJSON(w, http.StatusOK, map[string]any{"summary": map[string]any{}})
		return
	}
	writeJSON(w, http.StatusOK, s.SDK.RiskPrio.SummarizeAssessment(id))
}

func (s *Server) handleAssetRisks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.RiskPrio == nil {
		writeJSON(w, http.StatusOK, map[string]any{"risks": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"risks": s.SDK.RiskPrio.ListByAsset(contracts.AssetID(id))})
}

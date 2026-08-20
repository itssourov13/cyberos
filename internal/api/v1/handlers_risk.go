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
	project := r.URL.Query().Get("project_id")
	assessment := r.URL.Query().Get("assessment_id")
	if assessment != "" {
		writeJSON(w, http.StatusOK, map[string]any{"risks": s.SDK.RiskPrio.ListByAssessment(assessment)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"risks": s.SDK.RiskPrio.ListByProject(contracts.ProjectID(project))})
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

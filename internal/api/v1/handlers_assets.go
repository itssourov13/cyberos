package v1

import (
	"net/http"

	"cyberos/internal/contracts"
)

func (s *Server) handleAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.Assets == nil {
		writeJSON(w, http.StatusOK, map[string]any{"assets": []any{}})
		return
	}
	project := r.URL.Query().Get("project_id")
	writeJSON(w, http.StatusOK, map[string]any{"assets": s.SDK.Assets.ListByProject(contracts.ProjectID(project))})
}

func (s *Server) handleAssetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := r.PathValue("id")
	if s.SDK == nil || s.SDK.Assets == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	a, err := s.SDK.Assets.Get(contracts.AssetID(id))
	if err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, a)
}

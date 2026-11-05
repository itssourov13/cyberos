package v1

import (
	"errors"
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
	project, ok := requireProjectScope(w, r)
	if !ok {
		return
	}
	if project == "" {
		writeJSON(w, http.StatusOK, map[string]any{"assets": []any{}})
		return
	}
	assets, err := s.SDK.Assets.ListByProjectPage(r.Context(), project, 500, 0)
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": assets})
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
	if errors.Is(err, contracts.ErrNotFound) || (err == nil && a == nil) {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeErr(w, statusForErr(err), publicErr(err))
		return
	}
	if !enforceObject(w, r, "", a.ProjectID) {
		return
	}
	writeJSON(w, http.StatusOK, a)
}

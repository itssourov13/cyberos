package v1

import (
	"net/http"

	"cyberos/internal/contracts"
)

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.SDK == nil || s.SDK.Catalog == nil {
		writeJSON(w, http.StatusOK, map[string]any{"projects": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": s.SDK.Catalog.ListProjects()})
}

func (s *Server) handleScopes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := r.URL.Query().Get("project_id")
	if s.SDK == nil || s.SDK.Catalog == nil {
		writeJSON(w, http.StatusOK, map[string]any{"scopes": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"scopes": s.SDK.Catalog.ListScopes(contracts.ProjectID(projectID))})
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	projectID := r.URL.Query().Get("project_id")
	scopeID := r.URL.Query().Get("scope_id")
	if s.SDK == nil || s.SDK.Catalog == nil {
		writeJSON(w, http.StatusOK, map[string]any{"tasks": []any{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": s.SDK.Catalog.ListTasks(contracts.ProjectID(projectID), contracts.ScopeID(scopeID))})
}

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	// Path: /api/v1/tasks/{id}
	id := r.PathValue("id")
	if id == "" {
		writeErr(w, http.StatusBadRequest, "task id required")
		return
	}
	if s.SDK == nil || s.SDK.Catalog == nil {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	t, ok := s.SDK.Catalog.GetTask(contracts.TaskID(id))
	if !ok {
		writeErr(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var projects, scopes, tasks, caps int
	if s.SDK != nil {
		if s.SDK.Catalog != nil {
			projects = len(s.SDK.Catalog.ListProjects())
			scopes = len(s.SDK.Catalog.ListScopes(""))
			tasks = len(s.SDK.Catalog.ListTasks("", ""))
		}
		if s.SDK.Caps != nil {
			list, _ := s.SDK.ListCapabilities(r.Context())
			caps = len(list)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"projects":     projects,
		"scopes":       scopes,
		"tasks":        tasks,
		"capabilities": caps,
		"security":     "Scope → Principal → Control Plane → Grant → Broker → Sandbox",
	})
}

package v1

import (
	"net/http"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

func callerTenant(r *http.Request) contracts.TenantID {
	t, _ := auth.TenantFromContext(r.Context())
	return t
}

func requestedProject(r *http.Request) contracts.ProjectID {
	if p := r.URL.Query().Get("project_id"); p != "" {
		return contracts.ProjectID(p)
	}
	return contracts.ProjectID(r.Header.Get("X-CyberOS-Project-ID"))
}

// enforceTenant applies AssertSameTenant when the resource carries a tenant id.
// Project-only resources (empty resource tenant) skip this check; project scope still applies.
func enforceTenant(w http.ResponseWriter, r *http.Request, resourceTenant contracts.TenantID) bool {
	if resourceTenant == "" {
		return true
	}
	if err := auth.AssertSameTenant(callerTenant(r), resourceTenant); err != nil {
		writeErr(w, http.StatusForbidden, err.Error())
		return false
	}
	return true
}

func enforceProject(w http.ResponseWriter, r *http.Request, resourceProject contracts.ProjectID) bool {
	req := requestedProject(r)
	ct := callerTenant(r)
	if ct != "" {
		if req == "" {
			writeErr(w, http.StatusForbidden, "project scope required")
			return false
		}
		if resourceProject != "" && req != resourceProject {
			writeErr(w, http.StatusForbidden, "cross-project access denied")
			return false
		}
		return true
	}
	if req != "" && resourceProject != "" && req != resourceProject {
		writeErr(w, http.StatusForbidden, "cross-project access denied")
		return false
	}
	return true
}

func enforceResource(w http.ResponseWriter, r *http.Request, resourceTenant contracts.TenantID, resourceProject contracts.ProjectID) bool {
	if !enforceTenant(w, r, resourceTenant) {
		return false
	}
	return enforceProject(w, r, resourceProject)
}

func requireProjectScope(w http.ResponseWriter, r *http.Request) (contracts.ProjectID, bool) {
	p := requestedProject(r)
	if callerTenant(r) != "" && p == "" {
		writeErr(w, http.StatusForbidden, "project_id required for tenant-scoped list")
		return "", false
	}
	return p, true
}

// bindPrincipal resolves the authoritative principal for a request.
//
// When authentication is enabled the authenticated identity from the request
// context is the only accepted identity: a body-supplied principal that differs
// from it is rejected (403) and an omitted one is filled in from the context.
// When authentication is disabled (explicit development mode) the requested
// value is passed through unchanged.
func (s *Server) bindPrincipal(w http.ResponseWriter, r *http.Request, requested contracts.PrincipalID) (contracts.PrincipalID, bool) {
	if p, ok := auth.PrincipalFromContext(r.Context()); ok {
		if requested != "" && requested != p.ID {
			writeErr(w, http.StatusForbidden, "principal_id does not match the authenticated principal")
			return "", false
		}
		return p.ID, true
	}
	if s.Auth != nil {
		// Auth is enabled but no identity reached the handler: fail closed.
		writeErr(w, http.StatusUnauthorized, "authentication required")
		return "", false
	}
	return requested, true
}

// requireGrantOwner ensures the caller presenting a grant is the principal it was issued to.
// In development mode without auth the check is skipped (same as bindPrincipal).
func (s *Server) requireGrantOwner(w http.ResponseWriter, r *http.Request, grant contracts.ExecutionGrant) bool {
	if _, ok := auth.PrincipalFromContext(r.Context()); ok && grant.Principal.ID == "" {
		writeErr(w, http.StatusForbidden, "grant has no principal")
		return false
	}
	_, ok := s.bindPrincipal(w, r, grant.Principal.ID)
	return ok
}

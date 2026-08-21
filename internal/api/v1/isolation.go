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

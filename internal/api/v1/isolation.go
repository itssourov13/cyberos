package v1

import (
	"context"
	"net/http"

	"cyberos/internal/auth"
	"cyberos/internal/contracts"
)

// Authorization model (Phase 24D-1)
//
// Ownership is always resolved from authoritative server-side records:
//
//   - the caller's tenant comes from the authenticated identity (never a body field),
//   - the owning tenant of a project comes from the project catalog,
//   - the owning project of an assessment/case/finding/asset/risk comes from the
//     stored resource itself.
//
// A project id supplied in a query string, header or JSON body is only ever a
// *claim* that is checked against those records. It is never proof of ownership.
// Tenant-scoped callers therefore no longer need to send a project header for
// body-addressed writes (grants, execute, assessments, workflows, autonomy): the
// project named by the validated body/resource is resolved and checked instead.
// If a header/query project is also present it must agree with the resource.

type ctxProjectOwnerKey struct{}

// projectOwnerFunc resolves the owning tenant of a project from authoritative data.
type projectOwnerFunc func(contracts.ProjectID) (contracts.TenantID, bool)

// withProjectOwner installs the authoritative project-owner resolver for the
// request. It runs for every API request regardless of whether authentication
// is enabled so enforcement helpers never depend on handler-local wiring.
func (s *Server) withProjectOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var fn projectOwnerFunc = func(contracts.ProjectID) (contracts.TenantID, bool) { return "", false }
		if s.SDK != nil && s.SDK.Catalog != nil {
			fn = s.SDK.Catalog.ProjectTenant
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxProjectOwnerKey{}, fn)))
	})
}

// projectOwner resolves a project's owning tenant. Without an installed
// resolver nothing is known, so tenant-scoped callers fail closed.
func projectOwner(r *http.Request, p contracts.ProjectID) (contracts.TenantID, bool) {
	fn, ok := r.Context().Value(ctxProjectOwnerKey{}).(projectOwnerFunc)
	if !ok || fn == nil {
		return "", false
	}
	return fn(p)
}

func callerTenant(r *http.Request) contracts.TenantID {
	t, _ := auth.TenantFromContext(r.Context())
	return t
}

// requestedProject returns the optional client-declared project (query/header).
// It is a claim to be verified, not an authority.
func requestedProject(r *http.Request) contracts.ProjectID {
	if p := r.URL.Query().Get("project_id"); p != "" {
		return contracts.ProjectID(p)
	}
	return contracts.ProjectID(r.Header.Get("X-CyberOS-Project-ID"))
}

// authorizeProject is the single decision point for project access. It returns
// (0, "") when access is allowed, otherwise an HTTP status and message.
func authorizeProject(r *http.Request, resourceProject contracts.ProjectID) (int, string) {
	req := requestedProject(r)
	if req != "" && resourceProject != "" && req != resourceProject {
		return http.StatusForbidden, "cross-project access denied"
	}
	ct := callerTenant(r)
	if ct == "" {
		// Platform operator / legacy single-tenant credential.
		return 0, ""
	}
	effective := resourceProject
	if effective == "" {
		effective = req
	}
	if effective == "" {
		return http.StatusForbidden, "project scope required"
	}
	owner, known := projectOwner(r, effective)
	if !known || owner != ct {
		return http.StatusForbidden, "project not accessible"
	}
	return 0, ""
}

// authorizeTenant applies AssertSameTenant when the resource carries a tenant id.
func authorizeTenant(r *http.Request, resourceTenant contracts.TenantID) (int, string) {
	if resourceTenant == "" {
		return 0, ""
	}
	if err := auth.AssertSameTenant(callerTenant(r), resourceTenant); err != nil {
		return http.StatusForbidden, err.Error()
	}
	return 0, ""
}

// enforceTenant applies AssertSameTenant when the resource carries a tenant id.
func enforceTenant(w http.ResponseWriter, r *http.Request, resourceTenant contracts.TenantID) bool {
	if st, msg := authorizeTenant(r, resourceTenant); st != 0 {
		writeErr(w, st, msg)
		return false
	}
	return true
}

// enforceProject authorizes access to the named project (see authorizeProject).
func enforceProject(w http.ResponseWriter, r *http.Request, resourceProject contracts.ProjectID) bool {
	if st, msg := authorizeProject(r, resourceProject); st != 0 {
		writeErr(w, st, msg)
		return false
	}
	return true
}

// enforceResource authorizes a resource by its stored tenant and project.
func enforceResource(w http.ResponseWriter, r *http.Request, resourceTenant contracts.TenantID, resourceProject contracts.ProjectID) bool {
	if !enforceTenant(w, r, resourceTenant) {
		return false
	}
	return enforceProject(w, r, resourceProject)
}

// enforceObject authorizes an ID-addressed resource. A denial is reported as
// 404 so callers cannot probe for the existence of other tenants' identifiers.
func enforceObject(w http.ResponseWriter, r *http.Request, resourceTenant contracts.TenantID, resourceProject contracts.ProjectID) bool {
	if st, _ := authorizeTenant(r, resourceTenant); st != 0 {
		writeErr(w, http.StatusNotFound, "not found")
		return false
	}
	if st, _ := authorizeProject(r, resourceProject); st != 0 {
		writeErr(w, http.StatusNotFound, "not found")
		return false
	}
	return true
}

// resourceVisible reports (without writing a response) whether the caller may
// see an object owned by the given project. Used to filter list results.
func resourceVisible(r *http.Request, resourceProject contracts.ProjectID) bool {
	ct := callerTenant(r)
	if ct == "" {
		return true
	}
	if resourceProject == "" {
		return false
	}
	owner, known := projectOwner(r, resourceProject)
	return known && owner == ct
}

// requireProjectScope returns the project a list request is scoped to. For
// tenant-scoped callers the project must be named AND owned by the tenant.
func requireProjectScope(w http.ResponseWriter, r *http.Request) (contracts.ProjectID, bool) {
	p := requestedProject(r)
	if callerTenant(r) != "" {
		if p == "" {
			writeErr(w, http.StatusForbidden, "project_id required for tenant-scoped list")
			return "", false
		}
		if st, msg := authorizeProject(r, p); st != 0 {
			writeErr(w, st, msg)
			return "", false
		}
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

// projectGuard verifies, for tenant-scoped callers, that a project they
// declared via query/header belongs to their tenant. Per-resource checks in
// authorizeProject are authoritative; this is defense in depth that rejects a
// bad claim before any handler runs.
func (s *Server) projectGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.TenantFromContext(r.Context()); ok {
			if p := requestedProject(r); p != "" {
				if st, _ := authorizeProject(r, p); st != 0 {
					writeErr(w, http.StatusForbidden, "project not accessible")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requireOperator restricts platform-wide administration to tenant-less
// identities (the bootstrap/operator key). When auth is disabled (explicit dev
// mode) it is a no-op.
func (s *Server) requireOperator(w http.ResponseWriter, r *http.Request) bool {
	if s.Auth == nil {
		return true
	}
	if _, ok := auth.PrincipalFromContext(r.Context()); !ok {
		writeErr(w, http.StatusUnauthorized, "authentication required")
		return false
	}
	if t, ok := auth.TenantFromContext(r.Context()); ok && t != "" {
		writeErr(w, http.StatusForbidden, "platform administration requires an operator credential")
		return false
	}
	return true
}

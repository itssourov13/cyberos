package auth

import (
	"context"
	"strings"
)

// Role is a coarse permission level bound to a credential. Roles never grant
// cross-tenant access; tenant/project ownership is enforced separately.
type Role string

const (
	// RoleViewer may read resources inside its tenant/project.
	RoleViewer Role = "viewer"
	// RoleAnalyst may additionally create and operate on resources (cases,
	// assessments, grants, executions, workflows).
	RoleAnalyst Role = "analyst"
	// RoleAdmin may additionally perform sensitive transitions (risk acceptance,
	// worker administration, autonomy approval).
	RoleAdmin Role = "admin"
	// RoleOperator is the platform operator (tenant-less credential); it is the
	// only role that may administer platform-wide state (trust, capabilities).
	RoleOperator Role = "operator"
)

// Permission names a sensitive capability checked by the API layer.
type Permission string

const (
	PermRead            Permission = "read"
	PermWrite           Permission = "write"
	PermCaseTransition  Permission = "case.transition"
	PermCaseAccept      Permission = "case.accept"
	PermExecute         Permission = "execute"
	PermWorkerAdmin     Permission = "worker.admin"
	PermAutonomyControl Permission = "autonomy.control"
	PermAutonomyApprove Permission = "autonomy.approve"
	PermTrustAdmin      Permission = "trust.admin"
	PermAuditRead       Permission = "audit.read"
)

// ParseRole normalizes a known role. Unknown values fail closed with an
// empty role; storage adapters must reject them instead of silently escalating.
func ParseRole(s string) Role {
	switch Role(strings.ToLower(strings.TrimSpace(s))) {
	case RoleViewer:
		return RoleViewer
	case RoleAnalyst:
		return RoleAnalyst
	case RoleAdmin:
		return RoleAdmin
	case RoleOperator:
		return RoleOperator
	default:
		return ""
	}
}

// ValidRole reports whether s is a known role name.
func ValidRole(s string) bool {
	switch Role(strings.ToLower(strings.TrimSpace(s))) {
	case RoleViewer, RoleAnalyst, RoleAdmin, RoleOperator:
		return true
	}
	return false
}

// Allows reports whether the role carries the permission.
func (r Role) Allows(p Permission) bool {
	switch r {
	case RoleOperator:
		return true
	case RoleAdmin:
		return p != PermTrustAdmin
	case RoleAnalyst:
		switch p {
		case PermRead, PermWrite, PermCaseTransition, PermExecute, PermAutonomyControl, PermAuditRead:
			return true
		}
		return false
	case RoleViewer:
		return p == PermRead
	}
	return false
}

type ctxRoleKey struct{}

// RoleFromContext returns the authenticated role. A tenant-less identity is the
// platform operator regardless of any stored role string.
func RoleFromContext(ctx context.Context) Role {
	if _, ok := TenantFromContext(ctx); !ok {
		if _, pok := PrincipalFromContext(ctx); pok {
			return RoleOperator
		}
		return ""
	}
	r, _ := ctx.Value(ctxRoleKey{}).(Role)
	if r == "" || r == RoleOperator {
		// A missing or impossible tenant-scoped role fails closed. Legacy rows
		// are normalized by storage adapters before the identity reaches here.
		return ""
	}
	return r
}

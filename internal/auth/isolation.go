package auth

import (
	"fmt"

	"cyberos/internal/contracts"
)

// ResourceRef is a minimal ownership descriptor for isolation checks.
type ResourceRef struct {
	TenantID  contracts.TenantID
	ProjectID contracts.ProjectID
}

// AssertSameTenant rejects cross-tenant access.
// Authentication must already have bound the caller's tenant.
// Empty resource tenant is treated as legacy (allowed only when caller has no tenant).
func AssertSameTenant(callerTenant, resourceTenant contracts.TenantID) error {
	if callerTenant == "" && resourceTenant == "" {
		return nil // single-tenant / legacy mode
	}
	if callerTenant == "" || resourceTenant == "" {
		return fmt.Errorf("%w: tenant isolation requires both caller and resource tenant", contracts.ErrForbidden)
	}
	if callerTenant != resourceTenant {
		return fmt.Errorf("%w: cross-tenant access denied", contracts.ErrForbidden)
	}
	return nil
}

// AssertProjectTenant ensures a project belongs to the caller's tenant.
func AssertProjectTenant(callerTenant, projectTenant contracts.TenantID, projectID contracts.ProjectID) error {
	if err := AssertSameTenant(callerTenant, projectTenant); err != nil {
		return fmt.Errorf("%w: project %s", err, projectID)
	}
	return nil
}

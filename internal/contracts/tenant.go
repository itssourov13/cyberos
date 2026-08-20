package contracts

// TenantID identifies an organization/tenant boundary.
type TenantID string

// TenantStatus is the lifecycle of a tenant.
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "ACTIVE"
	TenantStatusSuspended TenantStatus = "SUSPENDED"
	TenantStatusDeleted   TenantStatus = "DELETED"
)

// Valid reports whether the tenant status is known.
func (s TenantStatus) Valid() bool {
	switch s {
	case TenantStatusActive, TenantStatusSuspended, TenantStatusDeleted:
		return true
	default:
		return false
	}
}

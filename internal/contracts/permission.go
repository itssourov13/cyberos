package contracts

// PermissionRef is a lightweight reference to a permission.
type PermissionRef struct {
	ID PermissionID `json:"id"`
}

// Validate checks required fields.
func (p PermissionRef) Validate() error {
	return ValidateNonEmpty("permission.id", string(p.ID))
}

// PermissionResult captures the outcome of a permission check.
type PermissionResult struct {
	PermissionID PermissionID `json:"permission_id,omitempty"`
	Granted      bool         `json:"granted"`
	Reason       string       `json:"reason,omitempty"`
}

package contracts

// ProjectRef is a lightweight reference to a project.
type ProjectRef struct {
	ID ProjectID `json:"id"`
}

// Validate checks required fields.
func (p ProjectRef) Validate() error {
	return ValidateNonEmpty("project.id", string(p.ID))
}

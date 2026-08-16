package contracts

// PolicyRef is a lightweight reference to a policy.
type PolicyRef struct {
	ID PolicyID `json:"id"`
}

// Validate checks required fields.
func (p PolicyRef) Validate() error {
	return ValidateNonEmpty("policy.id", string(p.ID))
}

// PolicyResult captures the outcome of a policy evaluation.
type PolicyResult struct {
	PolicyID PolicyID `json:"policy_id,omitempty"`
	Allowed  bool     `json:"allowed"`
	Reason   string   `json:"reason,omitempty"`
}

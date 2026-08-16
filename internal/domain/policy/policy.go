// Package policy provides domain helpers around policy results.
// Full policy evaluation belongs to the Control Plane (later phase).
package policy

import "cyberos/internal/contracts"

// Result is an alias for the contract shape used inside the domain.
type Result = contracts.PolicyResult

// Allowed is a convenience constructor for an allow result.
func Allowed(policyID contracts.PolicyID, reason string) Result {
	return Result{PolicyID: policyID, Allowed: true, Reason: reason}
}

// Denied is a convenience constructor for a deny result.
func Denied(policyID contracts.PolicyID, reason string) Result {
	return Result{PolicyID: policyID, Allowed: false, Reason: reason}
}

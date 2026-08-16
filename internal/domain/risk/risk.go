// Package risk provides domain helpers around risk results.
// Full risk evaluation belongs to the Control Plane (later phase).
package risk

import "cyberos/internal/contracts"

// Result is an alias for the contract shape used inside the domain.
type Result = contracts.RiskResult

// Accept constructs an accepted risk result.
func Accept(level contracts.RiskLevel, reason string) Result {
	return Result{Level: level, Accepted: true, Reason: reason}
}

// Reject constructs a rejected risk result.
func Reject(level contracts.RiskLevel, reason string) Result {
	return Result{Level: level, Accepted: false, Reason: reason}
}

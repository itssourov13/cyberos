// Package permission provides domain helpers around permission results.
// Full permission evaluation belongs to the Control Plane (later phase).
package permission

import "cyberos/internal/contracts"

// Result is an alias for the contract shape used inside the domain.
type Result = contracts.PermissionResult

// Granted is a convenience constructor.
func Granted(id contracts.PermissionID, reason string) Result {
	return Result{PermissionID: id, Granted: true, Reason: reason}
}

// Denied is a convenience constructor.
func Denied(id contracts.PermissionID, reason string) Result {
	return Result{PermissionID: id, Granted: false, Reason: reason}
}

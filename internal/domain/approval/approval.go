// Package approval provides domain helpers around approval results.
// Full approval workflow belongs to the Control Plane (later phase).
package approval

import "cyberos/internal/contracts"

// Result is an alias for the contract shape used inside the domain.
type Result = contracts.ApprovalResult

// Approved constructs an approved result.
func Approved(id contracts.ApprovalID, approver *contracts.PrincipalRef, reason string) Result {
	return Result{
		ApprovalID: id,
		Status:     contracts.ApprovalStatusApproved,
		Reason:     reason,
		Approver:   approver,
	}
}

// Denied constructs a denied result.
func Denied(id contracts.ApprovalID, reason string) Result {
	return Result{
		ApprovalID: id,
		Status:     contracts.ApprovalStatusDenied,
		Reason:     reason,
	}
}

// Pending constructs a pending result.
func Pending(id contracts.ApprovalID) Result {
	return Result{
		ApprovalID: id,
		Status:     contracts.ApprovalStatusPending,
	}
}

// Skipped constructs a skipped (not required) result.
func Skipped(reason string) Result {
	return Result{
		Status: contracts.ApprovalStatusSkipped,
		Reason: reason,
	}
}

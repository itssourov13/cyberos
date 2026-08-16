package contracts

import "errors"

// Stable, domain-neutral error categories.
// Concrete error messages may be wrapped, but callers should use errors.Is
// against these sentinels for control flow.

var (
	// ErrInvalidArgument indicates a caller-supplied value is malformed or missing.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrNotFound indicates a requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrConflict indicates a state conflict (e.g. duplicate, concurrent update).
	ErrConflict = errors.New("conflict")

	// ErrUnauthorized indicates authentication failed or is missing.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden indicates the principal is authenticated but not permitted.
	ErrForbidden = errors.New("forbidden")

	// ErrScopeInvalid indicates the scope is missing, expired, inactive, or otherwise unusable.
	ErrScopeInvalid = errors.New("scope invalid")

	// ErrGrantInvalid indicates an execution grant is missing, expired, malformed, or mismatched.
	ErrGrantInvalid = errors.New("execution grant invalid")

	// ErrGrantExpired indicates the grant's expiration time has passed.
	ErrGrantExpired = errors.New("execution grant expired")

	// ErrCapabilityUnknown indicates the requested capability is not registered.
	ErrCapabilityUnknown = errors.New("capability unknown")

	// ErrPolicyDenied indicates a policy evaluation denied the request.
	ErrPolicyDenied = errors.New("policy denied")

	// ErrPermissionDenied indicates the required permission is not held.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrRiskRejected indicates the risk evaluation rejected the request.
	ErrRiskRejected = errors.New("risk rejected")

	// ErrApprovalRequired indicates human or policy approval is still required.
	ErrApprovalRequired = errors.New("approval required")

	// ErrApprovalDenied indicates the required approval was denied.
	ErrApprovalDenied = errors.New("approval denied")

	// ErrInvalidStateTransition indicates an illegal task or entity state change.
	ErrInvalidStateTransition = errors.New("invalid state transition")

	// ErrNotImplemented is used for deliberately unimplemented contract paths.
	ErrNotImplemented = errors.New("not implemented")
)

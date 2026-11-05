package v1

import (
	"errors"
	"net/http"

	"cyberos/internal/contracts"
)

// statusForErr maps domain errors to HTTP statuses. Unknown errors are 500.
func statusForErr(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, contracts.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, contracts.ErrConflict):
		return http.StatusConflict
	case errors.Is(err, contracts.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, contracts.ErrForbidden),
		errors.Is(err, contracts.ErrPermissionDenied),
		errors.Is(err, contracts.ErrPolicyDenied),
		errors.Is(err, contracts.ErrApprovalRequired),
		errors.Is(err, contracts.ErrApprovalDenied),
		errors.Is(err, contracts.ErrRiskRejected),
		errors.Is(err, contracts.ErrScopeInvalid):
		return http.StatusForbidden
	case errors.Is(err, contracts.ErrInvalidArgument),
		errors.Is(err, contracts.ErrInvalidStateTransition),
		errors.Is(err, contracts.ErrCapabilityUnknown):
		return http.StatusBadRequest
	case errors.Is(err, contracts.ErrNotImplemented):
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

// publicErr returns a message that is safe to show to API clients. Domain
// (client-class) errors keep their message; anything else is replaced with a
// generic string so storage/driver details never leak.
func publicErr(err error) string {
	if err == nil {
		return ""
	}
	if statusForErr(err) >= http.StatusInternalServerError {
		return "internal error"
	}
	return err.Error()
}

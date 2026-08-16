// Package scope provides application services that treat Scope as the
// mandatory root authorization boundary for CYBEROS.
//
// A task cannot receive authorization outside a valid, active scope.
// Expired, inactive, or revoked scopes fail RequireActive.
package scope

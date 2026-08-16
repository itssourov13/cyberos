// Package identity provides principal (identity) management services.
//
// Principals are first-class actors. Before any authorization decision,
// the identity service must confirm the principal exists and is active.
//
// Dependency direction:
//
//	contracts → domain/principal → identity (this package) → storage/ports
package identity

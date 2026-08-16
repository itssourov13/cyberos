package contracts

import (
	"fmt"
	"time"
)

// Typed identifiers. Using distinct types prevents accidental mixing of IDs
// at compile time while remaining simple string wrappers for now.
// Future phases may evolve the underlying representation without breaking
// the public shape of these types.

type (
	ProjectID        string
	AssetID          string
	ScopeID          string
	TaskID           string
	PrincipalID      string
	CapabilityID     string
	PolicyID         string
	PermissionID     string
	ApprovalID       string
	ExecutionGrantID string
	EvidenceID       string
	FindingID        string
	EventID          string
	AuditID          string
	DecisionID       string
	CorrelationID    string
	CausationID      string
	SchemaVersion    string
)

// Common time helpers used across contracts.
// All timestamps in contracts are UTC.
type Timestamp = time.Time

// NowUTC returns the current time in UTC. Prefer this over time.Now() in
// contract construction to keep behavior consistent.
func NowUTC() Timestamp {
	return time.Now().UTC()
}

// ValidateNonEmpty is a small helper used by constructors.
func ValidateNonEmpty(field, value string) error {
	if value == "" {
		return fmt.Errorf("%w: %s must not be empty", ErrInvalidArgument, field)
	}
	return nil
}

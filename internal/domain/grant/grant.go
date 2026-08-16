// Package grant implements the ExecutionGrant domain entity.
//
// Invariant (enforced structurally; issuance belongs to later phases):
//
//	NO VALID GRANT  ⇒  NO EXECUTION
package grant

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Grant is the domain representation of an ExecutionGrant.
// It wraps and validates the contract shape while adding domain helpers.
type Grant struct {
	contracts.ExecutionGrant
}

// NewFromContract constructs a domain Grant from a contract value after validation.
func NewFromContract(g contracts.ExecutionGrant) (*Grant, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return &Grant{ExecutionGrant: g}, nil
}

// Validate delegates to the contract and adds domain-level checks.
func (g *Grant) Validate() error {
	if g == nil {
		return fmt.Errorf("%w: grant is nil", contracts.ErrInvalidArgument)
	}
	return g.ExecutionGrant.Validate()
}

// IsUsable reports whether the grant can be consumed at the given time.
func (g *Grant) IsUsable(now time.Time) bool {
	if g == nil {
		return false
	}
	return g.ExecutionGrant.IsUsable(now)
}

// MatchesContext verifies that the grant is bound to the expected
// principal, project, scope, task, and capability. Mismatches must
// cause the Tool Broker (later phase) to reject the grant.
func (g *Grant) MatchesContext(
	principalID contracts.PrincipalID,
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
	taskID contracts.TaskID,
	capabilityID contracts.CapabilityID,
) bool {
	if g == nil {
		return false
	}
	return g.Principal.ID == principalID &&
		g.ProjectID == projectID &&
		g.ScopeID == scopeID &&
		g.TaskID == taskID &&
		g.CapabilityID == capabilityID
}

// EnsureUsable returns nil if the grant is usable and matches context;
// otherwise it returns a suitable sentinel error.
func (g *Grant) EnsureUsable(
	now time.Time,
	principalID contracts.PrincipalID,
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
	taskID contracts.TaskID,
	capabilityID contracts.CapabilityID,
) error {
	if g == nil {
		return contracts.ErrGrantInvalid
	}
	if err := g.Validate(); err != nil {
		return fmt.Errorf("%w: %v", contracts.ErrGrantInvalid, err)
	}
	if !g.IsUsable(now) {
		if g.Status != contracts.ExecutionGrantStatusIssued {
			return contracts.ErrGrantInvalid
		}
		return contracts.ErrGrantExpired
	}
	if !g.MatchesContext(principalID, projectID, scopeID, taskID, capabilityID) {
		return contracts.ErrGrantInvalid
	}
	return nil
}

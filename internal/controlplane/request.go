package controlplane

import (
	"time"

	"cyberos/internal/contracts"
)

// EvaluateRequest is the input to a Control Plane evaluation.
type EvaluateRequest struct {
	PrincipalID   contracts.PrincipalID
	ProjectID     contracts.ProjectID
	ScopeID       contracts.ScopeID
	TaskID        contracts.TaskID
	CapabilityID  contracts.CapabilityID
	AssetID       contracts.AssetID // optional; checked against scope constraints when set
	CorrelationID contracts.CorrelationID
	// Now overrides the evaluation clock (useful for tests). Zero means time.Now().
	Now time.Time
	// RequireApproval forces an approval check (default policy may skip).
	RequireApproval bool
	// RiskLevelHint is an optional caller-supplied risk hint for evaluation.
	RiskLevelHint contracts.RiskLevel
}

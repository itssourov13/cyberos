// Package autonomy implements Phase 13A Observe → Analyze (no execution).
package autonomy

import (
	"time"

	"cyberos/internal/contracts"
)

// AnalysisContractV1 is the AnalysisResult contract version.
const AnalysisContractV1 contracts.SchemaVersion = "1"

const TrustUntrusted = "untrusted"

// ObservationRequest is the authoritative boundary for analysis (caller-controlled).
type ObservationRequest struct {
	ProjectID     contracts.ProjectID     `json:"project_id"`
	TenantID      contracts.TenantID      `json:"tenant_id,omitempty"`
	ScopeID       contracts.ScopeID       `json:"scope_id,omitempty"`
	PrincipalID   contracts.PrincipalID   `json:"principal_id,omitempty"`
	CorrelationID contracts.CorrelationID `json:"correlation_id,omitempty"`
	CausationID   contracts.CausationID   `json:"causation_id,omitempty"`
	Objective     string                  `json:"objective"`
	// SourceRef is an opaque reference to observation set (not secrets).
	SourceRef string `json:"source_ref,omitempty"`
	// ProviderID/Model are optional metadata for provenance (not credentials).
	ProviderID string `json:"provider_id,omitempty"`
	Model      string `json:"model,omitempty"`
}

// AnalysisItem is one untrusted model-derived item.
type AnalysisItem struct {
	ID      string `json:"id,omitempty"`
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
	Trust   string `json:"trust"` // always untrusted
}

// AnalysisResult is untrusted AI analysis data only — not a grant or workflow.
type AnalysisResult struct {
	ContractVersion         contracts.SchemaVersion  `json:"contract_version"`
	AnalysisID              string                   `json:"analysis_id"`
	ProjectID               contracts.ProjectID      `json:"project_id"`
	TenantID                contracts.TenantID       `json:"tenant_id,omitempty"`
	ScopeID                 contracts.ScopeID        `json:"scope_id,omitempty"`
	PrincipalID             contracts.PrincipalID    `json:"principal_id,omitempty"`
	CorrelationID           contracts.CorrelationID  `json:"correlation_id,omitempty"`
	CausationID             contracts.CausationID    `json:"causation_id,omitempty"`
	CreatedAt               time.Time                `json:"created_at"`
	Summary                 string                   `json:"summary"`
	Trust                   string                   `json:"trust"`
	Items                   []AnalysisItem           `json:"items,omitempty"`
	RecommendedCapabilities []contracts.CapabilityID `json:"recommended_capabilities,omitempty"`
	Confidence              string                   `json:"confidence,omitempty"`
	ProviderID              string                   `json:"provider_id,omitempty"`
	Model                   string                   `json:"model,omitempty"`
	SourceRef               string                   `json:"source_ref,omitempty"`
	Warnings                []string                 `json:"warnings,omitempty"`
}

// Limits for analysis results.
type Limits struct {
	MaxSummaryLen   int
	MaxItems        int
	MaxItemLen      int
	MaxCapabilities int
}

// DefaultLimits returns conservative bounds.
func DefaultLimits() Limits {
	return Limits{MaxSummaryLen: 4096, MaxItems: 50, MaxItemLen: 1024, MaxCapabilities: 20}
}

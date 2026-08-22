// Package ai defines the provider-neutral Brain contract and Plan IR (Phase 12A).
// AI output is untrusted. Plans never authorize or execute.
package ai

import (
	"cyberos/internal/contracts"
)

// PlanContractV1 is the supported Plan IR contract version.
const PlanContractV1 contracts.SchemaVersion = "1"

// Plan is the intermediate representation between AI reasoning and workflow definitions.
// It targets capabilities only — never executables, argv, or shell commands.
type Plan struct {
	ID              string                  `json:"id"`
	ContractVersion contracts.SchemaVersion `json:"contract_version"`
	Goal            string                  `json:"goal"`
	ProjectID       contracts.ProjectID     `json:"project_id"`
	ScopeID         contracts.ScopeID       `json:"scope_id"`
	TenantID        contracts.TenantID      `json:"tenant_id,omitempty"`
	PrincipalID     contracts.PrincipalID   `json:"principal_id,omitempty"`
	CorrelationID   contracts.CorrelationID `json:"correlation_id,omitempty"`
	CausationID     contracts.CausationID   `json:"causation_id,omitempty"`
	Steps           []PlanStep              `json:"steps"`
	Constraints     map[string]string       `json:"constraints,omitempty"`
	Rationale       string                  `json:"rationale,omitempty"`
}

// PlanStep is one capability-oriented step in a Plan.
type PlanStep struct {
	ID           string                 `json:"id"`
	CapabilityID contracts.CapabilityID `json:"capability_id"`
	Parameters   map[string]any         `json:"parameters,omitempty"`
	DependsOn    []string               `json:"depends_on,omitempty"`
	Rationale    string                 `json:"rationale,omitempty"`
}

// Analysis is a provider-neutral analysis result (data only).
type Analysis struct {
	Summary string            `json:"summary"`
	Details map[string]string `json:"details,omitempty"`
}

// Explanation is a provider-neutral explanation result (data only).
type Explanation struct {
	Text string `json:"text"`
}

// Operation identifies a Brain request kind.
type Operation string

const (
	OpPlan    Operation = "plan"
	OpAnalyze Operation = "analyze"
	OpExplain Operation = "explain"
)

// BrainRequest is a provider-neutral request into the Brain.
// It must not embed secrets or vendor SDK types.
type BrainRequest struct {
	Operation     Operation               `json:"operation"`
	PrincipalID   contracts.PrincipalID   `json:"principal_id"`
	ProjectID     contracts.ProjectID     `json:"project_id"`
	ScopeID       contracts.ScopeID       `json:"scope_id"`
	TenantID      contracts.TenantID      `json:"tenant_id,omitempty"`
	Objective     string                  `json:"objective"`
	CorrelationID contracts.CorrelationID `json:"correlation_id,omitempty"`
	// CapabilityHints lists capability IDs available for planning (discovery context).
	CapabilityHints []contracts.CapabilityID `json:"capability_hints,omitempty"`
	// ContextRef is an opaque reference to evidence/context (not raw secrets).
	ContextRef string `json:"context_ref,omitempty"`
}

// BrainResponse is a provider-neutral normalized response.
type BrainResponse struct {
	Status      string       `json:"status"` // ok | error
	Plan        *Plan        `json:"plan,omitempty"`
	Analysis    *Analysis    `json:"analysis,omitempty"`
	Explanation *Explanation `json:"explanation,omitempty"`
	Error       string       `json:"error,omitempty"`
	// ProviderMeta is non-authoritative metadata (must not drive execution).
	ProviderMeta map[string]string `json:"provider_meta,omitempty"`
}

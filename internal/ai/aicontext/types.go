// Package aicontext builds AI-safe, isolated, redacted context for Brain providers.
// Data only — never authorizes or executes. Phase 12B.
package aicontext

import (
	"cyberos/internal/contracts"
)

// ContractVersion for AIContext documents.
const ContractV1 contracts.SchemaVersion = "1"

// TrustLevel marks data origin for the model boundary.
type TrustLevel string

const (
	TrustControl   TrustLevel = "control"   // CYBEROS metadata / capability catalog
	TrustUntrusted TrustLevel = "untrusted" // tool output, external text, findings body
)

// ContextRequest defines the isolation boundary for assembly.
type ContextRequest struct {
	TenantID      contracts.TenantID      `json:"tenant_id,omitempty"`
	ProjectID     contracts.ProjectID     `json:"project_id"`
	ScopeID       contracts.ScopeID       `json:"scope_id,omitempty"`
	PrincipalID   contracts.PrincipalID   `json:"principal_id,omitempty"`
	CorrelationID contracts.CorrelationID `json:"correlation_id,omitempty"`
	// Include selectors (all optional; empty = assemble available sections).
	IncludeCapabilities bool `json:"include_capabilities,omitempty"`
	IncludeTools        bool `json:"include_tools,omitempty"`
	IncludeSchemas      bool `json:"include_schemas,omitempty"`
	IncludeFindings     bool `json:"include_findings,omitempty"`
	IncludeEvidence     bool `json:"include_evidence,omitempty"`
	IncludeWorkflow     bool `json:"include_workflow,omitempty"`
}

// Limits bounds context size (byte-oriented).
type Limits struct {
	MaxTotalBytes      int
	MaxSectionBytes    int
	MaxItemBytes       int
	MaxItemsPerSection int
}

// DefaultLimits returns conservative defaults.
func DefaultLimits() Limits {
	return Limits{
		MaxTotalBytes:      256 * 1024,
		MaxSectionBytes:    64 * 1024,
		MaxItemBytes:       8 * 1024,
		MaxItemsPerSection: 100,
	}
}

// Section is one labeled context section.
type Section struct {
	Name      string     `json:"name"`
	Trust     TrustLevel `json:"trust"`
	Items     []Item     `json:"items,omitempty"`
	Truncated bool       `json:"truncated,omitempty"`
}

// Item is one context entry.
type Item struct {
	ID      string         `json:"id,omitempty"`
	Trust   TrustLevel     `json:"trust"`
	Fields  map[string]any `json:"fields,omitempty"`
	Content string         `json:"content,omitempty"`
}

// AIContext is the serializable AI-safe context document.
type AIContext struct {
	ContractVersion contracts.SchemaVersion `json:"contract_version"`
	TenantID        contracts.TenantID      `json:"tenant_id,omitempty"`
	ProjectID       contracts.ProjectID     `json:"project_id"`
	ScopeID         contracts.ScopeID       `json:"scope_id,omitempty"`
	PrincipalID     contracts.PrincipalID   `json:"principal_id,omitempty"`
	CorrelationID   contracts.CorrelationID `json:"correlation_id,omitempty"`
	Sections        []Section               `json:"sections"`
	Warnings        []string                `json:"warnings,omitempty"`
	Truncated       bool                    `json:"truncated,omitempty"`
}

// SourceData is optional in-memory input for assembly (tests and wiring).
// All lists must already be isolation-filtered by the caller OR Assembler filters by ProjectID.
type SourceData struct {
	Capabilities []CapabilityInfo
	Tools        []ToolInfo
	Findings     []FindingInfo
	Evidence     []EvidenceInfo
	Workflow     []WorkflowInfo
}

// CapabilityInfo is catalog metadata (control trust).
type CapabilityInfo struct {
	ID      string
	Name    string
	Project contracts.ProjectID // empty = global catalog entry
}

// ToolInfo is tool/pack metadata (control trust).
type ToolInfo struct {
	ID           string
	Name         string
	Capabilities []string
	Project      contracts.ProjectID
}

// FindingInfo is finding summary (body untrusted).
type FindingInfo struct {
	ID        string
	Title     string
	Severity  string
	ProjectID contracts.ProjectID
	TenantID  contracts.TenantID
	Summary   string
}

// EvidenceInfo is evidence summary (content untrusted).
type EvidenceInfo struct {
	ID        string
	ProjectID contracts.ProjectID
	TenantID  contracts.TenantID
	Summary   string
	Content   string
}

// WorkflowInfo is workflow run metadata (control).
type WorkflowInfo struct {
	ID        string
	State     string
	ProjectID contracts.ProjectID
}

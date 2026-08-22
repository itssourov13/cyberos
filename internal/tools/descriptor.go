// Package tools is the security-tool adapter layer.
//
// Tools are registered as pluggable adapters. Discovery and health checks
// never authorize execution. Execution always goes through
// Tool Broker → Sandbox → Executor under a valid grant.
package tools

import (
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/tools/schema"
)

// HealthStatus reports adapter readiness.
type HealthStatus string

const (
	HealthUnknown   HealthStatus = "UNKNOWN"
	HealthHealthy   HealthStatus = "HEALTHY"
	HealthDegraded  HealthStatus = "DEGRADED"
	HealthUnhealthy HealthStatus = "UNHEALTHY"
)

// Descriptor is tool metadata independent of any specific product.
type Descriptor struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Version      contracts.SchemaVersion  `json:"version"`
	Runtime      contracts.PluginRuntime  `json:"runtime"`
	Capabilities []contracts.CapabilityID `json:"capabilities"`
	ExecutorID   string                   `json:"executor_id"`
	Description  string                   `json:"description,omitempty"`
	// Binary is optional host binary name for process-backed adapters (allowlisted separately).
	Binary string `json:"binary,omitempty"`
	// ContractVersion versions the tool extension contract; empty defaults to V1.
	ContractVersion contracts.SchemaVersion `json:"contract_version,omitempty"`
	// Schemas are optional declarative parameter/argv/output schemas (Phase 11B).
	// Discovery/validation only — never authorization.
	Schemas *schema.ToolSchemas `json:"schemas,omitempty"`
}

// Validate checks required descriptor fields.
func (d Descriptor) Validate() error {
	m := d.ToolManifest()
	if err := m.Validate(); err != nil {
		return err
	}
	if d.Schemas != nil {
		if err := d.Schemas.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ToolManifest maps descriptor to contracts.ToolManifest.
func (d Descriptor) ToolManifest() contracts.ToolManifest {
	return contracts.ToolManifest{
		ID:              d.ID,
		Name:            d.Name,
		Version:         d.Version,
		Runtime:         d.Runtime,
		Capabilities:    d.Capabilities,
		ContractVersion: d.ContractVersion,
		ExecutorID:      d.ExecutorID,
		Description:     d.Description,
	}
}

// HealthReport is the result of a health check.
type HealthReport struct {
	Status    HealthStatus `json:"status"`
	Message   string       `json:"message,omitempty"`
	CheckedAt time.Time    `json:"checked_at"`
}

// NormalizedResult is a tool-agnostic execution outcome for evidence/findings.
type NormalizedResult struct {
	ToolID     string         `json:"tool_id"`
	Capability string         `json:"capability"`
	Success    bool           `json:"success"`
	Summary    string         `json:"summary,omitempty"`
	Findings   []FindingHint  `json:"findings,omitempty"`
	Metrics    map[string]any `json:"metrics,omitempty"`
	Stdout     string         `json:"stdout,omitempty"`
	Stderr     string         `json:"stderr,omitempty"`
	Raw        map[string]any `json:"raw,omitempty"`
}

// FindingHint is a lightweight finding suggestion (not yet a domain Finding).
type FindingHint struct {
	Title    string `json:"title"`
	Severity string `json:"severity,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

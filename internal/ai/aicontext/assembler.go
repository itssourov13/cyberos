package aicontext

import (
	"fmt"
	"sort"

	"cyberos/internal/contracts"
)

// Assembler builds AIContext from SourceData under isolation and size limits.
type Assembler struct {
	Limits Limits
}

// NewAssembler constructs an assembler with default limits.
func NewAssembler() *Assembler {
	return &Assembler{Limits: DefaultLimits()}
}

// Assemble produces a deterministic, redacted AIContext.
// Does not authorize, execute, resolve secrets, or call network.
func (a *Assembler) Assemble(req ContextRequest, src SourceData) (*AIContext, error) {
	if a == nil {
		a = NewAssembler()
	}
	if req.ProjectID == "" {
		return nil, fmt.Errorf("%w: project_id required for context assembly", contracts.ErrInvalidArgument)
	}
	lim := a.Limits
	if lim.MaxItemsPerSection <= 0 {
		lim = DefaultLimits()
	}

	ctx := &AIContext{
		ContractVersion: ContractV1,
		TenantID:        req.TenantID,
		ProjectID:       req.ProjectID,
		ScopeID:         req.ScopeID,
		PrincipalID:     req.PrincipalID,
		CorrelationID:   req.CorrelationID,
	}

	includeAll := !req.IncludeCapabilities && !req.IncludeTools && !req.IncludeSchemas &&
		!req.IncludeFindings && !req.IncludeEvidence && !req.IncludeWorkflow

	if includeAll || req.IncludeCapabilities {
		sec := buildCapabilitySection(req, src.Capabilities, lim)
		ctx.Sections = append(ctx.Sections, sec)
	}
	if includeAll || req.IncludeTools {
		sec := buildToolSection(req, src.Tools, lim)
		ctx.Sections = append(ctx.Sections, sec)
	}
	if includeAll || req.IncludeFindings {
		sec, warn := buildFindingSection(req, src.Findings, lim)
		ctx.Sections = append(ctx.Sections, sec)
		ctx.Warnings = append(ctx.Warnings, warn...)
	}
	if includeAll || req.IncludeEvidence {
		sec, warn := buildEvidenceSection(req, src.Evidence, lim)
		ctx.Sections = append(ctx.Sections, sec)
		ctx.Warnings = append(ctx.Warnings, warn...)
	}
	if includeAll || req.IncludeWorkflow {
		sec := buildWorkflowSection(req, src.Workflow, lim)
		ctx.Sections = append(ctx.Sections, sec)
	}

	// Sort sections by name for determinism
	sort.Slice(ctx.Sections, func(i, j int) bool {
		return ctx.Sections[i].Name < ctx.Sections[j].Name
	})

	for _, s := range ctx.Sections {
		if s.Truncated {
			ctx.Truncated = true
		}
	}
	return ctx, nil
}

func projectOK(req ContextRequest, project contracts.ProjectID) bool {
	if project == "" {
		return true // global catalog
	}
	return project == req.ProjectID
}

func tenantOK(req ContextRequest, tenant contracts.TenantID) bool {
	if req.TenantID == "" || tenant == "" {
		return true
	}
	return tenant == req.TenantID
}

func buildCapabilitySection(req ContextRequest, caps []CapabilityInfo, lim Limits) Section {
	sec := Section{Name: "capabilities", Trust: TrustControl}
	var items []Item
	for _, c := range caps {
		if !projectOK(req, c.Project) {
			continue
		}
		items = append(items, Item{
			ID: c.ID, Trust: TrustControl,
			Fields: map[string]any{"id": c.ID, "name": c.Name},
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if len(items) > lim.MaxItemsPerSection {
		items = items[:lim.MaxItemsPerSection]
		sec.Truncated = true
	}
	sec.Items = items
	return sec
}

func buildToolSection(req ContextRequest, tools []ToolInfo, lim Limits) Section {
	sec := Section{Name: "tools", Trust: TrustControl}
	var items []Item
	for _, t := range tools {
		if !projectOK(req, t.Project) {
			continue
		}
		caps := append([]string{}, t.Capabilities...)
		sort.Strings(caps)
		items = append(items, Item{
			ID: t.ID, Trust: TrustControl,
			Fields: map[string]any{"id": t.ID, "name": t.Name, "capabilities": caps},
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if len(items) > lim.MaxItemsPerSection {
		items = items[:lim.MaxItemsPerSection]
		sec.Truncated = true
	}
	sec.Items = items
	return sec
}

func buildFindingSection(req ContextRequest, findings []FindingInfo, lim Limits) (Section, []string) {
	sec := Section{Name: "findings", Trust: TrustUntrusted}
	var warn []string
	var items []Item
	for _, f := range findings {
		if !projectOK(req, f.ProjectID) || !tenantOK(req, f.TenantID) {
			continue
		}
		sum, trunc := Truncate(RedactText(f.Summary), lim.MaxItemBytes)
		if trunc {
			sec.Truncated = true
		}
		items = append(items, Item{
			ID: f.ID, Trust: TrustUntrusted,
			Fields:  RedactFields(map[string]any{"title": f.Title, "severity": f.Severity}),
			Content: sum,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if len(items) > lim.MaxItemsPerSection {
		items = items[:lim.MaxItemsPerSection]
		sec.Truncated = true
		warn = append(warn, "findings truncated by item limit")
	}
	sec.Items = items
	return sec, warn
}

func buildEvidenceSection(req ContextRequest, ev []EvidenceInfo, lim Limits) (Section, []string) {
	sec := Section{Name: "evidence", Trust: TrustUntrusted}
	var warn []string
	var items []Item
	for _, e := range ev {
		if !projectOK(req, e.ProjectID) || !tenantOK(req, e.TenantID) {
			continue
		}
		content, trunc := Truncate(RedactText(e.Content), lim.MaxItemBytes)
		if trunc {
			sec.Truncated = true
		}
		sum, _ := Truncate(RedactText(e.Summary), lim.MaxItemBytes)
		items = append(items, Item{
			ID: e.ID, Trust: TrustUntrusted,
			Fields:  map[string]any{"summary": sum},
			Content: content,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if len(items) > lim.MaxItemsPerSection {
		items = items[:lim.MaxItemsPerSection]
		sec.Truncated = true
		warn = append(warn, "evidence truncated by item limit")
	}
	sec.Items = items
	return sec, warn
}

func buildWorkflowSection(req ContextRequest, runs []WorkflowInfo, lim Limits) Section {
	sec := Section{Name: "workflow", Trust: TrustControl}
	var items []Item
	for _, w := range runs {
		if !projectOK(req, w.ProjectID) {
			continue
		}
		items = append(items, Item{
			ID: w.ID, Trust: TrustControl,
			Fields: map[string]any{"id": w.ID, "state": w.State},
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if len(items) > lim.MaxItemsPerSection {
		items = items[:lim.MaxItemsPerSection]
		sec.Truncated = true
	}
	sec.Items = items
	return sec
}

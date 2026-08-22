package tools

import (
	"fmt"

	"cyberos/internal/contracts"
	"cyberos/internal/tools/schema"
)

// PackFormatV1 is the supported pack file encoding version.
const PackFormatV1 contracts.SchemaVersion = "1"

// PackFile is the on-disk/declarative pack package format (Phase 11C).
// Loading a PackFile is discovery only — never executes tools or changes allowlists.
type PackFile struct {
	FormatVersion contracts.SchemaVersion `json:"format_version"`
	Pack          Pack                    `json:"pack"`
	Tools         []PackTool              `json:"tools,omitempty"`
}

// PackTool is a declarative tool entry inside a pack file.
type PackTool struct {
	ID              string                   `json:"id"`
	Name            string                   `json:"name"`
	Version         contracts.SchemaVersion  `json:"version"`
	Runtime         contracts.PluginRuntime  `json:"runtime"`
	ExecutorID      string                   `json:"executor_id"`
	Description     string                   `json:"description,omitempty"`
	Binary          string                   `json:"binary,omitempty"`
	Capabilities    []contracts.CapabilityID `json:"capabilities,omitempty"`
	ContractVersion contracts.SchemaVersion  `json:"contract_version,omitempty"`
	Schemas         *schema.ToolSchemas      `json:"schemas,omitempty"`
}

// Validate checks pack file structure without executing anything.
func (f PackFile) Validate() error {
	fv := f.FormatVersion
	if fv == "" {
		fv = PackFormatV1
	}
	if fv != PackFormatV1 {
		return fmt.Errorf("%w: unsupported pack format version %q", contracts.ErrInvalidArgument, f.FormatVersion)
	}
	if err := f.Pack.Validate(); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for i, t := range f.Tools {
		d := t.Descriptor()
		if err := d.Validate(); err != nil {
			return fmt.Errorf("tools[%d]: %w", i, err)
		}
		if _, ok := seen[t.ID]; ok {
			return fmt.Errorf("%w: duplicate tool id %q", contracts.ErrInvalidArgument, t.ID)
		}
		seen[t.ID] = struct{}{}
	}
	if len(f.Pack.ToolIDs) > 0 && len(f.Tools) > 0 {
		for _, id := range f.Pack.ToolIDs {
			if _, ok := seen[id]; !ok {
				return fmt.Errorf("%w: pack tool_ids references unknown tool %q", contracts.ErrInvalidArgument, id)
			}
		}
	}
	return nil
}

// Descriptor maps declarative tool metadata (discovery only).
func (t PackTool) Descriptor() Descriptor {
	exec := t.ExecutorID
	if exec == "" {
		exec = t.ID
	}
	rt := t.Runtime
	if rt == "" {
		rt = contracts.PluginRuntimeGo
	}
	return Descriptor{
		ID: t.ID, Name: t.Name, Version: t.Version, Runtime: rt,
		ExecutorID: exec, Description: t.Description, Binary: t.Binary,
		Capabilities: t.Capabilities, ContractVersion: t.ContractVersion, Schemas: t.Schemas,
	}
}

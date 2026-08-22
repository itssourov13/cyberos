package workflow

import (
	"fmt"
	"sort"

	"cyberos/internal/contracts"
	"cyberos/internal/tools"
)

// ResolvedTool is a discovery hit for a capability (never an authorization).
type ResolvedTool struct {
	ToolID       string
	ExecutorID   string
	CapabilityID contracts.CapabilityID
	PackID       string
	PackVersion  contracts.SchemaVersion
}

// CapabilityResolver maps capability → eligible registered tools.
// It never executes tools or issues grants.
type CapabilityResolver struct {
	Platform *tools.Platform
}

// NewCapabilityResolver constructs a resolver over a tool platform.
func NewCapabilityResolver(p *tools.Platform) *CapabilityResolver {
	return &CapabilityResolver{Platform: p}
}

// Resolve returns eligible tools for a capability, sorted deterministically by ToolID.
func (r *CapabilityResolver) Resolve(cap contracts.CapabilityID) ([]ResolvedTool, error) {
	if r == nil || r.Platform == nil {
		return nil, fmt.Errorf("%w: capability resolver not configured", contracts.ErrInvalidArgument)
	}
	if cap == "" {
		return nil, fmt.Errorf("%w: capability id required", contracts.ErrInvalidArgument)
	}
	// Reject capabilities from disabled/deprecated packs
	disabledTools := map[string]struct{}{}
	for _, pack := range r.Platform.Packs().List() {
		if pack.State == tools.PackStateDisabled || pack.State == tools.PackStateDeprecated {
			for _, tid := range pack.ToolIDs {
				disabledTools[tid] = struct{}{}
			}
		}
	}
	var out []ResolvedTool
	for _, m := range r.Platform.LookupCapability(cap) {
		if _, bad := disabledTools[m.ToolID]; bad {
			continue
		}
		// If pack is disabled we already skipped; also skip empty tool
		if m.ToolID == "" {
			continue
		}
		out = append(out, ResolvedTool{
			ToolID: m.ToolID, ExecutorID: m.ExecutorID, CapabilityID: m.CapabilityID,
			PackID: m.PackID, PackVersion: m.Version,
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no eligible tool for capability %s", contracts.ErrNotFound, cap)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ToolID < out[j].ToolID })
	return out, nil
}

// ResolvePrimary returns the deterministic primary tool for a capability.
func (r *CapabilityResolver) ResolvePrimary(cap contracts.CapabilityID) (ResolvedTool, error) {
	all, err := r.Resolve(cap)
	if err != nil {
		return ResolvedTool{}, err
	}
	return all[0], nil
}

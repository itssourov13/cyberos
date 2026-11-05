package v1

import (
	"fmt"
	"strings"

	"cyberos/internal/contracts"
)

// resolveExecutorHint derives a unique registered executor for a capability.
// A caller-supplied hint is accepted only when it is registered and implements
// that capability; the grant signature then binds the resolved executor.
func (s *Server) resolveExecutorHint(capability contracts.CapabilityID, requested string) (string, error) {
	if s.SDK == nil || s.SDK.ExecReg == nil {
		if s.SDK != nil && s.SDK.RequireExplicitExecutorHint {
			return "", fmt.Errorf("%w: executor registry unavailable", contracts.ErrCapabilityUnknown)
		}
		return "", nil
	}
	manifests := s.SDK.ExecReg.ListManifests()
	if requested != "" {
		for _, m := range manifests {
			if (m.ExecutorID == requested || m.ID == requested) && hasCapability(m.Capabilities, capability) {
				return m.ExecutorID, nil
			}
		}
		return "", fmt.Errorf("%w: executor hint is not registered for capability %s", contracts.ErrCapabilityUnknown, capability)
	}
	var candidates []string
	for _, m := range manifests {
		if hasCapability(m.Capabilities, capability) {
			candidates = append(candidates, m.ExecutorID)
		}
	}
	switch len(candidates) {
	case 0:
		if s.SDK.RequireExplicitExecutorHint {
			return "", fmt.Errorf("%w: no production executor is registered for capability %s", contracts.ErrCapabilityUnknown, capability)
		}
		return "", nil
	case 1:
		return candidates[0], nil
	default:
		return "", fmt.Errorf("%w: capability %s has multiple executors; an explicit executor_hint is required", contracts.ErrInvalidArgument, capability)
	}
}

func hasCapability(items []contracts.CapabilityID, target contracts.CapabilityID) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func canonicalTargetMatch(input, canonical string) bool {
	return input == "" || strings.TrimSpace(input) == canonical
}

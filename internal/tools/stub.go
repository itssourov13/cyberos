package tools

import (
	"context"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/execution/process"
)

// StubScanner is a non-offensive demonstration adapter used for tests and
// architecture validation. It does not run real security scanners.
// Real product adapters (Nmap, Nuclei, …) should follow the same Adapter shape.
type StubScanner struct {
	proc *process.Runner
	desc Descriptor
}

// NewStubScanner constructs the demo security-assessment adapter.
func NewStubScanner(proc *process.Runner) *StubScanner {
	return &StubScanner{
		proc: proc,
		desc: Descriptor{
			ID:           "stub.security.scanner",
			Name:         "Stub Security Scanner",
			Version:      "1",
			Runtime:      contracts.PluginRuntimeGo,
			Capabilities: []contracts.CapabilityID{"assessment.stub.probe"},
			ExecutorID:   "stub.security.scanner",
			Description:  "In-process stub for adapter-layer validation (not a real scanner)",
		},
	}
}

// Descriptor implements Adapter.
func (s *StubScanner) Descriptor() Descriptor { return s.desc }

// Health implements Adapter.
func (s *StubScanner) Health(ctx context.Context) HealthReport {
	_ = ctx
	return HealthReport{Status: HealthHealthy, Message: "stub ready", CheckedAt: time.Now().UTC()}
}

// Run implements Adapter — pure in-process normalized result; optional process path still fail-closed.
func (s *StubScanner) Run(ctx context.Context, capability contracts.CapabilityID, params map[string]any) (NormalizedResult, error) {
	start := time.Now()
	if capability != "assessment.stub.probe" {
		return NormalizedResult{}, fmt.Errorf("%w: unknown capability", contracts.ErrCapabilityUnknown)
	}
	// If caller requests process execution, enforce process policy (still fail-closed by default).
	if _, ok := params["process"]; ok && s.proc != nil {
		_, _, err := s.proc.Run(ctx, process.Spec{Path: fmt.Sprint(params["path"])})
		if err != nil {
			return NormalizedResult{
				ToolID: s.desc.ID, Capability: string(capability), Success: false,
				Summary: "process path rejected", Metrics: map[string]any{"duration_ms": time.Since(start).Milliseconds()},
			}, err
		}
	}
	target, _ := params["target"].(string)
	if target == "" {
		target = "unspecified"
	}
	return NormalizedResult{
		ToolID:     s.desc.ID,
		Capability: string(capability),
		Success:    true,
		Summary:    "stub probe completed for " + target,
		Findings: []FindingHint{{
			Title:    "Stub observation",
			Severity: "INFO",
			Detail:   "Demonstration finding for target " + target,
		}},
		Metrics: map[string]any{
			"duration_ms": time.Since(start).Milliseconds(),
			"adapter":     "stub",
		},
		Raw: map[string]any{"target": target, "mode": "in-process"},
	}, nil
}

var _ Adapter = (*StubScanner)(nil)

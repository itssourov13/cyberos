package ai

import (
	"context"
	"fmt"

	"cyberos/internal/contracts"
)

// MockBrain is a deterministic, offline Brain for tests. No network, no execution.
type MockBrain struct {
	// FixedPlan, if set, is returned from Plan after validation of request.
	FixedPlan *Plan
}

// Plan returns a deterministic capability-oriented plan.
func (m *MockBrain) Plan(ctx context.Context, req BrainRequest) (BrainResponse, error) {
	if err := ValidateBrainRequest(req); err != nil {
		return BrainResponse{Status: "error", Error: err.Error()}, err
	}
	if req.Operation != OpPlan && req.Operation != "" {
		// allow OpPlan default
	}
	p := m.FixedPlan
	if p == nil {
		capID := contracts.CapabilityID("recon.network.portscan")
		if len(req.CapabilityHints) > 0 {
			capID = req.CapabilityHints[0]
		}
		p = &Plan{
			ID:              "plan-mock-1",
			ContractVersion: PlanContractV1,
			Goal:            req.Objective,
			ProjectID:       req.ProjectID,
			ScopeID:         req.ScopeID,
			TenantID:        req.TenantID,
			PrincipalID:     req.PrincipalID,
			CorrelationID:   req.CorrelationID,
			Steps: []PlanStep{{
				ID:           "step-1",
				CapabilityID: capID,
				Parameters:   map[string]any{"target": "example.com"},
				Rationale:    "mock deterministic step",
			}},
			Rationale: "mock brain",
		}
	}
	if err := ValidatePlan(p); err != nil {
		return BrainResponse{Status: "error", Error: err.Error()}, err
	}
	return BrainResponse{Status: "ok", Plan: p, ProviderMeta: map[string]string{"provider": "mock"}}, nil
}

// Analyze returns a deterministic analysis.
func (m *MockBrain) Analyze(ctx context.Context, req BrainRequest) (BrainResponse, error) {
	if err := ValidateBrainRequest(req); err != nil {
		return BrainResponse{Status: "error", Error: err.Error()}, err
	}
	return BrainResponse{
		Status:       "ok",
		Analysis:     &Analysis{Summary: fmt.Sprintf("mock analysis of: %s", req.Objective)},
		ProviderMeta: map[string]string{"provider": "mock"},
	}, nil
}

// Explain returns a deterministic explanation.
func (m *MockBrain) Explain(ctx context.Context, req BrainRequest) (BrainResponse, error) {
	if err := ValidateBrainRequest(req); err != nil {
		return BrainResponse{Status: "error", Error: err.Error()}, err
	}
	return BrainResponse{
		Status:       "ok",
		Explanation:  &Explanation{Text: fmt.Sprintf("mock explanation of: %s", req.Objective)},
		ProviderMeta: map[string]string{"provider": "mock"},
	}, nil
}

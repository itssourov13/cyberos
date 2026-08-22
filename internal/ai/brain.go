package ai

import "context"

// Brain is the provider-neutral planning/analysis contract.
// Implementations must not authorize, issue grants, or execute tools.
type Brain interface {
	Plan(ctx context.Context, req BrainRequest) (BrainResponse, error)
	Analyze(ctx context.Context, req BrainRequest) (BrainResponse, error)
	Explain(ctx context.Context, req BrainRequest) (BrainResponse, error)
}

// ToWorkflowShape returns a structural mapping of plan steps to capability + deps
// for future Phase 12D conversion. Does not create tasks, authorize, or execute.
func ToWorkflowShape(p *Plan) ([]struct {
	ID           string
	CapabilityID string
	DependsOn    []string
	Parameters   map[string]any
}, error) {
	if err := ValidatePlan(p); err != nil {
		return nil, err
	}
	out := make([]struct {
		ID           string
		CapabilityID string
		DependsOn    []string
		Parameters   map[string]any
	}, 0, len(p.Steps))
	for _, s := range p.Steps {
		out = append(out, struct {
			ID           string
			CapabilityID string
			DependsOn    []string
			Parameters   map[string]any
		}{
			ID: s.ID, CapabilityID: string(s.CapabilityID),
			DependsOn: append([]string{}, s.DependsOn...), Parameters: s.Parameters,
		})
	}
	return out, nil
}

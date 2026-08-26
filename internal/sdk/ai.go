package sdk

import (
	"context"
	"fmt"

	"cyberos/internal/ai"
	"cyberos/internal/ai/apply"
	"cyberos/internal/contracts"
	"cyberos/internal/workflow"
)

// Brain is an optional explicit Brain for external-brain API/SDK calls (Phase 15D).
// When nil, MockBrain is used. Never issues grants or calls Broker.
func (c *Client) WithBrain(b ai.Brain) *Client {
	c.Brain = b
	return c
}

// Analyze invokes Brain.Analyze (untrusted). No Grant/Broker.
func (c *Client) Analyze(ctx context.Context, req ai.BrainRequest) (ai.BrainResponse, error) {
	b := c.Brain
	if b == nil {
		b = &ai.MockBrain{}
	}
	req.Operation = ai.OpAnalyze
	return b.Analyze(ctx, req)
}

// Plan invokes Brain.Plan then ValidatePlan (untrusted plan). No execution.
func (c *Client) Plan(ctx context.Context, req ai.BrainRequest) (ai.BrainResponse, error) {
	b := c.Brain
	if b == nil {
		b = &ai.MockBrain{}
	}
	req.Operation = ai.OpPlan
	resp, err := b.Plan(ctx, req)
	if err != nil {
		return resp, err
	}
	if resp.Plan != nil {
		if vErr := ai.ValidatePlan(resp.Plan); vErr != nil {
			return ai.BrainResponse{Status: "error", Error: vErr.Error(), ProviderMeta: resp.ProviderMeta}, vErr
		}
	}
	return resp, nil
}

// ApplyPlan applies a validated plan to a Workflow Definition only (no execution).
func (c *Client) ApplyPlan(ctx context.Context, plan ai.Plan) (*workflow.Definition, error) {
	if c.AIApply == nil {
		c.AIApply = apply.NewApplicator(nil)
	}
	res, err := c.AIApply.Apply(ctx, apply.ApplyRequest{
		Plan: plan, ProjectID: plan.ProjectID, TenantID: plan.TenantID,
		ScopeID: plan.ScopeID, PrincipalID: plan.PrincipalID,
		CorrelationID: plan.CorrelationID, Accepted: true,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, fmt.Errorf("%w: empty apply result", contracts.ErrInvalidArgument)
	}
	def := res.Definition
	return &def, nil
}

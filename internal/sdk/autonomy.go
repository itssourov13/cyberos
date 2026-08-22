package sdk

import (
	"context"
	"fmt"

	"cyberos/internal/ai"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
	"cyberos/internal/workflow"
)

// WireAutonomy attaches Autonomy Controller + optional Handoff (Phase 14C).
func (c *Client) WireAutonomy(brain ai.Brain) *Client {
	if brain == nil {
		brain = &ai.MockBrain{}
	}
	c.Autonomy = autonomy.NewController(brain)
	c.AutonomyHandoff = &autonomy.AdvanceHandoff{
		Controller:   c.Autonomy,
		Orchestrator: c.Workflow,
	}
	return c
}

// StartAutonomy starts an autonomy run (no automatic execution beyond existing controller semantics).
func (c *Client) StartAutonomy(ctx context.Context, req autonomy.StartRequest) (*autonomy.AutonomyRun, error) {
	if c.Autonomy == nil {
		return nil, fmt.Errorf("%w: autonomy not configured", contracts.ErrInvalidArgument)
	}
	return c.Autonomy.Start(ctx, req)
}

// GetAutonomy returns an autonomy run by id (in-memory or after LoadFromRepo).
func (c *Client) GetAutonomy(runID string) (*autonomy.AutonomyRun, error) {
	if c.Autonomy == nil {
		return nil, fmt.Errorf("%w: autonomy not configured", contracts.ErrInvalidArgument)
	}
	return c.Autonomy.Get(runID)
}

// CancelAutonomy cancels a run (idempotent).
func (c *Client) CancelAutonomy(runID string) (*autonomy.AutonomyRun, error) {
	if c.Autonomy == nil {
		return nil, fmt.Errorf("%w: autonomy not configured", contracts.ErrInvalidArgument)
	}
	return c.Autonomy.Cancel(runID)
}

// ResumeAutonomy loads durable state into the controller (14A recovery).
func (c *Client) ResumeAutonomy(ctx context.Context, projectID contracts.ProjectID, runID string) (*autonomy.AutonomyRun, error) {
	if c.Autonomy == nil {
		return nil, fmt.Errorf("%w: autonomy not configured", contracts.ErrInvalidArgument)
	}
	return c.Autonomy.LoadFromRepo(ctx, projectID, runID)
}

// AdvanceAutonomy invokes AdvanceHandoff (policy → Orchestrator). No direct Broker/Grant.
func (c *Client) AdvanceAutonomy(ctx context.Context, req autonomy.HandoffRequest) (*autonomy.HandoffResult, error) {
	if c.AutonomyHandoff == nil {
		return nil, fmt.Errorf("%w: autonomy handoff not configured", contracts.ErrInvalidArgument)
	}
	return c.AutonomyHandoff.Execute(ctx, req)
}

// Ensure workflow type referenced for wire
var _ = workflow.Definition{}

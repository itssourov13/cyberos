package sdk

import (
	"context"
	"errors"
	"fmt"

	"cyberos/internal/assessment"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
)

// CreateAssessment creates and stores an assessment (CREATED).
func (c *Client) CreateAssessment(id, projectID, scopeID, principalID, title, target string) (*assessment.Assessment, error) {
	if c.Assessment == nil {
		return nil, fmt.Errorf("%w: assessment orchestrator not configured", contracts.ErrInvalidArgument)
	}
	if existing, err := c.Assessment.Get(contracts.AssessmentID(id)); err == nil && existing != nil {
		return nil, contracts.ErrConflict
	} else if err != nil && !errors.Is(err, contracts.ErrNotFound) {
		return nil, err
	}
	a, err := assessment.New(
		contracts.AssessmentID(id), contracts.ProjectID(projectID), contracts.ScopeID(scopeID),
		contracts.PrincipalID(principalID), title, target,
	)
	if err != nil {
		return nil, err
	}
	if err := c.Assessment.Save(a); err != nil {
		return nil, err
	}
	c.syncAssessmentCatalog(a)
	return a, nil
}

// PlanAssessment attaches sequential capability steps.
func (c *Client) PlanAssessment(id string, steps []contracts.AssessmentStep) (*assessment.Assessment, error) {
	if c.Assessment == nil {
		return nil, fmt.Errorf("%w: assessment orchestrator not configured", contracts.ErrInvalidArgument)
	}
	a, err := c.Assessment.Get(contracts.AssessmentID(id))
	if err != nil {
		return nil, err
	}
	if err := a.Plan(steps); err != nil {
		return nil, err
	}
	if err := c.Assessment.Save(a); err != nil {
		return nil, err
	}
	c.syncAssessmentCatalog(a)
	return a, nil
}

// RunAssessment executes planned steps with per-step Control Plane grants.
func (c *Client) RunAssessment(ctx context.Context, id string) (*assessment.Assessment, error) {
	if c.Assessment == nil {
		return nil, fmt.Errorf("%w: assessment orchestrator not configured", contracts.ErrInvalidArgument)
	}
	out, err := c.Assessment.Run(ctx, contracts.AssessmentID(id))
	if out != nil {
		c.syncAssessmentCatalog(out)
		// The assessment state event is informational. The underlying audit and
		// event records for decisions, grants and executions are durable.
		if c.Hub != nil {
			c.Hub.Publish(contracts.EventEnvelope{
				ID:            contracts.EventID(fmt.Sprintf("sse-asmt-%s-%d", id, contracts.NowUTC().UnixNano())),
				Type:          contracts.EventType("assessment.state"),
				SchemaVersion: contracts.EventSchemaV1,
				Timestamp:     contracts.NowUTC(),
				ProjectID:     out.ProjectID, ScopeID: out.ScopeID,
				CorrelationID: contracts.CorrelationID("assessment-" + id),
				Payload: map[string]any{
					"assessment_id": string(out.ID),
					"state":         string(out.State),
					"steps":         len(out.StepResults),
				},
			})
		}
	}
	return out, err
}

// ListAssessments returns a bounded project-scoped assessment list from the
// durable repository in production.
func (c *Client) ListAssessments(projectID contracts.ProjectID, limit, offset int) ([]*assessment.Assessment, error) {
	if c.Assessment == nil {
		return nil, fmt.Errorf("%w: assessment orchestrator not configured", contracts.ErrInvalidArgument)
	}
	return c.Assessment.ListByProject(projectID, limit, offset)
}

// GetAssessment returns a stored assessment.
func (c *Client) GetAssessment(id string) (*assessment.Assessment, error) {
	if c.Assessment == nil {
		return nil, fmt.Errorf("%w: assessment orchestrator not configured", contracts.ErrInvalidArgument)
	}
	return c.Assessment.Get(contracts.AssessmentID(id))
}

func (c *Client) syncAssessmentCatalog(a *assessment.Assessment) {
	if c.Catalog == nil || a == nil {
		return
	}
	c.Catalog.UpsertAssessment(AssessmentView{
		ID: string(a.ID), ProjectID: string(a.ProjectID), ScopeID: string(a.ScopeID),
		PrincipalID: string(a.PrincipalID), Title: a.Title, Target: a.Target,
		State: string(a.State), StepCount: len(a.Steps),
	})
}

// sdkAssessmentIssuer routes per-step assessment authorization through the
// same audited SDK grant path used by API and workflow execution.
type sdkAssessmentIssuer struct {
	client *Client
}

func (a sdkAssessmentIssuer) IssueForStep(ctx context.Context, run *assessment.Assessment, step contracts.AssessmentStep) (contracts.ExecutionGrant, error) {
	c := a.client
	if c == nil || run == nil || c.Control == nil || c.Issuer == nil {
		return contracts.ExecutionGrant{}, fmt.Errorf("%w: assessment authorization unavailable", contracts.ErrInvalidArgument)
	}
	if c.RequireExplicitExecutorHint && step.ExecutorHint == "" {
		return contracts.ExecutionGrant{}, fmt.Errorf("%w: assessment step must declare an executor hint", contracts.ErrCapabilityUnknown)
	}
	req := controlplane.EvaluateRequest{
		PrincipalID: run.PrincipalID, ProjectID: run.ProjectID, ScopeID: run.ScopeID,
		TaskID:        contracts.TaskID("assessment-" + string(run.ID) + "-" + step.ID),
		CapabilityID:  step.CapabilityID,
		CorrelationID: contracts.CorrelationID("assessment-" + string(run.ID) + "-" + step.ID),
	}
	constraints := contracts.ExecutionGrantConstraints{}
	if step.ExecutorHint != "" {
		constraints.AllowedExecutorHints = []string{step.ExecutorHint}
	}
	if run.Target != "" {
		constraints.AllowedTargets = []string{run.Target}
		if c.Assets != nil {
			asset, err := c.Assets.GetByValue(run.ProjectID, run.Target)
			if err == nil {
				req.AssetID = asset.ID
				constraints.AllowedTargets = []string{asset.Value}
			} else if !errors.Is(err, contracts.ErrNotFound) {
				return contracts.ExecutionGrant{}, fmt.Errorf("resolve assessment target: %w", err)
			}
		}
	}
	grant, _, err := c.IssueGrantWithConstraints(ctx, req, constraints)
	return grant, err
}

// sdkAssessmentExecutor delegates through Client.Execute so audit persistence
// and the broker remain mandatory for each assessment result.
type sdkAssessmentExecutor struct {
	client *Client
}

func (e sdkAssessmentExecutor) Execute(req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	if e.client == nil {
		return contracts.ExecutionResult{}, fmt.Errorf("%w: assessment execution client unavailable", contracts.ErrInvalidArgument)
	}
	return e.client.Execute(req.Grant, req.Parameters)
}

var _ assessment.GrantIssuer = sdkAssessmentIssuer{}
var _ assessment.ExecutorGateway = sdkAssessmentExecutor{}

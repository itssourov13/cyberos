package sdk

import (
	"context"
	"fmt"

	"cyberos/internal/assessment"
	"cyberos/internal/contracts"
)

// CreateAssessment creates and stores an assessment (CREATED).
func (c *Client) CreateAssessment(id, projectID, scopeID, principalID, title, target string) (*assessment.Assessment, error) {
	if c.Assessment == nil {
		return nil, fmt.Errorf("%w: assessment orchestrator not configured", contracts.ErrInvalidArgument)
	}
	// Never let a create overwrite an existing assessment (possibly another tenant's).
	if existing, gerr := c.Assessment.Get(contracts.AssessmentID(id)); gerr == nil && existing != nil {
		return nil, contracts.ErrConflict
	}
	a, err := assessment.New(
		contracts.AssessmentID(id),
		contracts.ProjectID(projectID),
		contracts.ScopeID(scopeID),
		contracts.PrincipalID(principalID),
		title, target,
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
		if c.Hub != nil {
			c.Hub.Publish(contracts.EventEnvelope{
				ID:            contracts.EventID("sse-asmt-" + id),
				Type:          contracts.EventType("assessment.state"),
				SchemaVersion: contracts.EventSchemaV1,
				Timestamp:     contracts.NowUTC(),
				ProjectID:     out.ProjectID,
				ScopeID:       out.ScopeID,
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
		ID:          string(a.ID),
		ProjectID:   string(a.ProjectID),
		ScopeID:     string(a.ScopeID),
		PrincipalID: string(a.PrincipalID),
		Title:       a.Title,
		Target:      a.Target,
		State:       string(a.State),
		StepCount:   len(a.Steps),
	})
}

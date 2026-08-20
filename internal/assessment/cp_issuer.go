package assessment

import (
	"context"
	"fmt"

	"cyberos/internal/broker"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
)

// ControlPlaneIssuer issues per-step grants only after a successful Control Plane decision.
type ControlPlaneIssuer struct {
	Control *controlplane.Service
	Issuer  *broker.Issuer
}

// IssueForStep evaluates authorization then issues a short-lived grant.
func (c ControlPlaneIssuer) IssueForStep(ctx context.Context, a *Assessment, step contracts.AssessmentStep) (contracts.ExecutionGrant, error) {
	if c.Control == nil || c.Issuer == nil {
		return contracts.ExecutionGrant{}, fmt.Errorf("%w: control plane issuer not configured", contracts.ErrInvalidArgument)
	}
	if a == nil {
		return contracts.ExecutionGrant{}, contracts.ErrInvalidArgument
	}
	taskID := contracts.TaskID("assessment-" + string(a.ID) + "-" + step.ID)
	corr := contracts.CorrelationID("assessment-" + string(a.ID) + "-" + step.ID)
	req := controlplane.EvaluateRequest{
		PrincipalID:   a.PrincipalID,
		ProjectID:     a.ProjectID,
		ScopeID:       a.ScopeID,
		TaskID:        taskID,
		CapabilityID:  step.CapabilityID,
		CorrelationID: corr,
	}
	g, d, err := c.Issuer.IssueAfterEvaluate(ctx, c.Control, req)
	if err != nil {
		return contracts.ExecutionGrant{}, err
	}
	if !d.Allowed() {
		return contracts.ExecutionGrant{}, fmt.Errorf("%w: %s", contracts.ErrForbidden, d.Reason)
	}
	if step.ExecutorHint != "" {
		g.Constraints.AllowedExecutorHints = []string{step.ExecutorHint}
	}
	return g.ExecutionGrant, nil
}

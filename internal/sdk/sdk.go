// Package sdk provides the in-process SDK foundation for CYBEROS.
// Future automation, plugins, agents, ZOYA/JARVIS, and UAB should depend on
// this surface (or the HTTP API), not on internal implementation packages.
package sdk

import (
	"context"
	"fmt"
	"time"

	appscope "cyberos/internal/application/scope"
	"cyberos/internal/audit"
	"cyberos/internal/broker"
	"cyberos/internal/capability"
	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	domaincap "cyberos/internal/domain/capability"
	"cyberos/internal/evidence"
	"cyberos/internal/execution/mock"
	"cyberos/internal/findings"
	"cyberos/internal/identity"
	"cyberos/internal/provenance"
	"cyberos/internal/taskengine"
)

// Client is the SDK entrypoint. Sensitive operations always go through
// Control Plane → Grant → Broker.
type Client struct {
	Identity   *identity.Service
	Scopes     *appscope.Service
	Caps       *capability.Registry
	Tasks      *taskengine.Engine
	Control    *controlplane.Service
	Issuer     *broker.Issuer
	Broker     *broker.Broker
	Evidence   *evidence.Service
	Findings   *findings.Service
	Provenance *provenance.Service
	Audit      *audit.Service
}

// NewInMemory constructs a client with mock broker, issuer, evidence stack,
// and in-memory capability registry. Wire Identity/Scopes/Control/Tasks for
// full authorization flows.
func NewInMemory() *Client {
	return &Client{
		Caps:       capability.NewRegistry(nil),
		Issuer:     broker.NewIssuer(5 * time.Minute),
		Broker:     broker.New(mock.New()),
		Evidence:   evidence.NewService(),
		Findings:   findings.NewService(evidence.NewService()),
		Provenance: provenance.NewService(),
	}
}

// WithIdentity sets the identity service.
func (c *Client) WithIdentity(s *identity.Service) *Client { c.Identity = s; return c }

// WithScopes sets the scope service.
func (c *Client) WithScopes(s *appscope.Service) *Client { c.Scopes = s; return c }

// WithControlPlane sets the control plane.
func (c *Client) WithControlPlane(s *controlplane.Service) *Client { c.Control = s; return c }

// WithTasks sets the task engine.
func (c *Client) WithTasks(e *taskengine.Engine) *Client { c.Tasks = e; return c }

// WithAudit sets the audit service.
func (c *Client) WithAudit(a *audit.Service) *Client { c.Audit = a; return c }

// WithEvidence shares an evidence service (and rebuilds findings on top).
func (c *Client) WithEvidence(e *evidence.Service) *Client {
	c.Evidence = e
	c.Findings = findings.NewService(e)
	return c
}

// Evaluate runs the Control Plane decision chain.
func (c *Client) Evaluate(ctx context.Context, req controlplane.EvaluateRequest) (contracts.Decision, error) {
	if c.Control == nil {
		return contracts.Decision{}, fmt.Errorf("%w: control plane not configured", contracts.ErrInvalidArgument)
	}
	d, err := c.Control.Evaluate(ctx, req)
	if err != nil {
		return d, err
	}
	if c.Audit != nil && req.CorrelationID != "" {
		_, _ = c.Audit.RecordDecision(d)
	}
	return d, nil
}

// IssueGrant evaluates then issues a grant only if allowed.
func (c *Client) IssueGrant(ctx context.Context, req controlplane.EvaluateRequest) (contracts.ExecutionGrant, contracts.Decision, error) {
	if c.Issuer == nil || c.Control == nil {
		return contracts.ExecutionGrant{}, contracts.Decision{}, fmt.Errorf("%w: issuer/control not configured", contracts.ErrInvalidArgument)
	}
	g, d, err := c.Issuer.IssueAfterEvaluate(ctx, c.Control, req)
	if err != nil {
		return contracts.ExecutionGrant{}, d, err
	}
	if c.Audit != nil {
		_, _ = c.Audit.RecordGrantIssued(g.ExecutionGrant)
	}
	return g.ExecutionGrant, d, nil
}

// Execute runs work through the Tool Broker under a grant.
func (c *Client) Execute(grant contracts.ExecutionGrant, params map[string]any) (contracts.ExecutionResult, error) {
	if c.Broker == nil {
		return contracts.ExecutionResult{}, fmt.Errorf("%w: broker not configured", contracts.ErrInvalidArgument)
	}
	res, err := c.Broker.Execute(contracts.ExecutionRequest{
		Grant: grant, Parameters: params, CorrelationID: grant.CorrelationID,
	})
	if c.Audit != nil {
		_, _ = c.Audit.RecordExecution(grant, res)
	}
	return res, err
}

// RegisterCapability registers a capability for discovery only.
func (c *Client) RegisterCapability(ctx context.Context, id, name string, typ contracts.CapabilityType) error {
	if c.Caps == nil {
		return fmt.Errorf("%w: capability registry not configured", contracts.ErrInvalidArgument)
	}
	cap, err := domaincap.New(contracts.CapabilityID(id), name, typ)
	if err != nil {
		return err
	}
	return c.Caps.Register(ctx, cap)
}

// ListCapabilities returns registered capability descriptors.
func (c *Client) ListCapabilities(ctx context.Context) ([]contracts.CapabilityDescriptor, error) {
	if c.Caps == nil {
		return nil, fmt.Errorf("%w: capability registry not configured", contracts.ErrInvalidArgument)
	}
	list, err := c.Caps.List(ctx, 100, 0)
	if err != nil {
		return nil, err
	}
	out := make([]contracts.CapabilityDescriptor, 0, len(list))
	for _, cap := range list {
		out = append(out, cap.Descriptor())
	}
	return out, nil
}

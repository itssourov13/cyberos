// Package broker implements ExecutionGrant issuance and the Tool Broker gateway.
package broker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/controlplane"
	domaingrant "cyberos/internal/domain/grant"
)

// Issuer issues short-lived ExecutionGrants only after a successful Control Plane decision.
type Issuer struct {
	// TTL is the default grant lifetime. Zero means 5 minutes.
	TTL time.Duration
	// Signer, when set, signs every issued grant so the Broker can prove
	// provenance. Production wiring always sets it.
	Signer *GrantSigner
}

// NewIssuer constructs a grant issuer.
func NewIssuer(ttl time.Duration) *Issuer {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &Issuer{TTL: ttl}
}

// IssueRequest carries the context needed to issue a grant.
type IssueRequest struct {
	Decision      contracts.Decision
	Principal     contracts.PrincipalRef
	ProjectID     contracts.ProjectID
	ScopeID       contracts.ScopeID
	TaskID        contracts.TaskID
	CapabilityID  contracts.CapabilityID
	CorrelationID contracts.CorrelationID
	// Now overrides the clock (tests).
	Now time.Time
	// Constraints optional.
	Constraints contracts.ExecutionGrantConstraints
	// Pack binding from server-resolved trust (Phase 19A).
	PackContentDigest string
	PackKeyID         string
}

// Issue creates an ExecutionGrant only if the Control Plane decision is ALLOWED.
// Any other outcome returns ErrForbidden / ErrGrantInvalid.
func (i *Issuer) Issue(_ context.Context, req IssueRequest) (*domaingrant.Grant, error) {
	if i == nil {
		return nil, fmt.Errorf("%w: issuer not configured", contracts.ErrInvalidArgument)
	}
	if !req.Decision.Allowed() {
		return nil, fmt.Errorf("%w: cannot issue grant for denied decision: %s",
			contracts.ErrForbidden, req.Decision.Reason)
	}
	if err := req.Decision.Validate(); err != nil {
		return nil, fmt.Errorf("%w: invalid decision: %v", contracts.ErrInvalidArgument, err)
	}

	// Consistency: decision context must match request context when present.
	if req.Decision.Principal.ID != "" && req.Principal.ID != "" &&
		req.Decision.Principal.ID != req.Principal.ID {
		return nil, fmt.Errorf("%w: principal mismatch between decision and request", contracts.ErrGrantInvalid)
	}
	if req.Decision.ScopeID != "" && req.ScopeID != "" && req.Decision.ScopeID != req.ScopeID {
		return nil, fmt.Errorf("%w: scope mismatch between decision and request", contracts.ErrGrantInvalid)
	}
	if req.Decision.CapabilityID != "" && req.CapabilityID != "" &&
		req.Decision.CapabilityID != req.CapabilityID {
		return nil, fmt.Errorf("%w: capability mismatch between decision and request", contracts.ErrGrantInvalid)
	}

	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	nonce, err := randomNonce(16)
	if err != nil {
		return nil, err
	}
	principal := req.Principal
	if principal.ID == "" {
		principal = req.Decision.Principal
	}
	scopeID := req.ScopeID
	if scopeID == "" {
		scopeID = req.Decision.ScopeID
	}
	taskID := req.TaskID
	if taskID == "" {
		taskID = req.Decision.TaskID
	}
	capID := req.CapabilityID
	if capID == "" {
		capID = req.Decision.CapabilityID
	}
	projectID := req.ProjectID
	if projectID == "" {
		projectID = req.Decision.ProjectID
	}

	g := contracts.ExecutionGrant{
		ID:                contracts.ExecutionGrantID("grant-" + nonce[:12]),
		Principal:         principal,
		ProjectID:         projectID,
		ScopeID:           scopeID,
		TaskID:            taskID,
		CapabilityID:      capID,
		Status:            contracts.ExecutionGrantStatusIssued,
		IssuedAt:          now,
		ExpiresAt:         now.Add(i.TTL),
		Constraints:       req.Constraints,
		CorrelationID:     req.CorrelationID,
		Nonce:             nonce,
		PackContentDigest: req.PackContentDigest,
		PackKeyID:         req.PackKeyID,
	}
	if g.CorrelationID == "" {
		g.CorrelationID = req.Decision.CorrelationID
	}
	if i.Signer != nil {
		sig, err := i.Signer.Sign(g)
		if err != nil {
			return nil, fmt.Errorf("sign grant: %w", err)
		}
		g.Signature = sig
	}
	return domaingrant.NewFromContract(g)
}

// IssueAfterEvaluate is a convenience: run Control Plane evaluation then issue.
func (i *Issuer) IssueAfterEvaluate(
	ctx context.Context,
	cp *controlplane.Service,
	evalReq controlplane.EvaluateRequest,
) (*domaingrant.Grant, contracts.Decision, error) {
	return i.IssueAfterEvaluateWith(ctx, cp, evalReq, contracts.ExecutionGrantConstraints{})
}

// IssueAfterEvaluateWith is IssueAfterEvaluate with grant constraints fixed at
// issuance time. Constraints must be supplied here, never patched onto an
// issued grant: the issuer signature covers them.
func (i *Issuer) IssueAfterEvaluateWith(
	ctx context.Context,
	cp *controlplane.Service,
	evalReq controlplane.EvaluateRequest,
	constraints contracts.ExecutionGrantConstraints,
) (*domaingrant.Grant, contracts.Decision, error) {
	d, err := cp.Evaluate(ctx, evalReq)
	if err != nil {
		return nil, d, err
	}
	if !d.Allowed() {
		return nil, d, fmt.Errorf("%w: %s", contracts.ErrForbidden, d.Reason)
	}
	issueReq := IssueRequest{
		Decision:      d,
		Principal:     d.Principal,
		ProjectID:     evalReq.ProjectID,
		ScopeID:       evalReq.ScopeID,
		TaskID:        evalReq.TaskID,
		CapabilityID:  evalReq.CapabilityID,
		CorrelationID: evalReq.CorrelationID,
		Now:           evalReq.Now,
		Constraints:   constraints,
	}
	if evalReq.RequireTrustedPack {
		if evalReq.EffectivePackTrust != "TRUSTED" || evalReq.PackContentDigest == "" {
			return nil, d, fmt.Errorf("%w: trusted-pack grant requires server-derived digest", contracts.ErrGrantInvalid)
		}
		issueReq.PackContentDigest = evalReq.PackContentDigest
		issueReq.PackKeyID = evalReq.PackKeyID
	}
	g, err := i.Issue(ctx, issueReq)
	return g, d, err
}

func randomNonce(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

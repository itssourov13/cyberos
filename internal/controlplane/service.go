package controlplane

import (
	"context"
	"fmt"
	"time"

	"cyberos/internal/application/scope"
	"cyberos/internal/contracts"
	"cyberos/internal/identity"
)

// Service is the Control Plane evaluation engine.
type Service struct {
	identity *identity.Service
	scopes   *scope.Service
	// Simple in-process evaluators (Phase 1E). Future phases may replace
	// these with pluggable policy engines.
	policyEval     PolicyEvaluator
	permissionEval PermissionEvaluator
	riskEval       RiskEvaluator
	approvalEval   ApprovalEvaluator
}

// Option configures the Control Plane service.
type Option func(*Service)

// WithPolicyEvaluator overrides the default policy evaluator.
func WithPolicyEvaluator(e PolicyEvaluator) Option {
	return func(s *Service) { s.policyEval = e }
}

// WithPermissionEvaluator overrides the default permission evaluator.
func WithPermissionEvaluator(e PermissionEvaluator) Option {
	return func(s *Service) { s.permissionEval = e }
}

// WithRiskEvaluator overrides the default risk evaluator.
func WithRiskEvaluator(e RiskEvaluator) Option {
	return func(s *Service) { s.riskEval = e }
}

// WithApprovalEvaluator overrides the default approval evaluator.
func WithApprovalEvaluator(e ApprovalEvaluator) Option {
	return func(s *Service) { s.approvalEval = e }
}

// NewService constructs a Control Plane service.
func NewService(idSvc *identity.Service, scopeSvc *scope.Service, opts ...Option) *Service {
	s := &Service{
		identity:       idSvc,
		scopes:         scopeSvc,
		policyEval:     DefaultPolicyEvaluator{},
		permissionEval: DefaultPermissionEvaluator{},
		riskEval:       DefaultRiskEvaluator{},
		approvalEval:   DefaultApprovalEvaluator{},
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Evaluate runs the non-bypassable security chain and returns a structured Decision.
// It never returns a bare boolean. Every denial is explicit and carries a reason.
func (s *Service) Evaluate(ctx context.Context, req EvaluateRequest) (contracts.Decision, error) {
	now := req.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	decisionID := contracts.DecisionID(fmt.Sprintf("dec-%d", now.UnixNano()))
	base := contracts.Decision{
		ID:            decisionID,
		Timestamp:     now,
		ProjectID:     req.ProjectID,
		ScopeID:       req.ScopeID,
		TaskID:        req.TaskID,
		CapabilityID:  req.CapabilityID,
		CorrelationID: req.CorrelationID,
	}

	// 1. Scope — mandatory authorization boundary
	if req.ScopeID == "" {
		return s.deny(base, "scope is required", nil, nil, nil, nil), nil
	}
	sc, err := s.scopes.RequireActive(ctx, req.ScopeID, now)
	if err != nil {
		return s.deny(base, fmt.Sprintf("scope rejected: %v", err), nil, nil, nil, nil), nil
	}
	if err := scope.EnsureSameProject(sc, req.ProjectID); err != nil && req.ProjectID != "" {
		return s.deny(base, fmt.Sprintf("scope project mismatch: %v", err), nil, nil, nil, nil), nil
	}
	if len(sc.Constraints.AllowedAssetIDs) > 0 && req.AssetID == "" {
		return s.deny(base, "asset_id is required by this scope", nil, nil, nil, nil), nil
	}
	if req.AssetID != "" {
		if err := s.scopes.EnsureAllowsAsset(sc, req.AssetID); err != nil {
			return s.deny(base, fmt.Sprintf("asset outside scope: %v", err), nil, nil, nil, nil), nil
		}
	}

	// 2. Principal
	if req.PrincipalID == "" {
		return s.deny(base, "principal is required", nil, nil, nil, nil), nil
	}
	p, err := s.identity.RequireActive(ctx, req.PrincipalID)
	if err != nil {
		return s.deny(base, fmt.Sprintf("principal rejected: %v", err), nil, nil, nil, nil), nil
	}
	base.Principal = p.Ref()

	// 3–4. Policy
	policyRes := s.policyEval.Evaluate(ctx, req, p.Ref(), sc)
	base.Policy = &policyRes
	if !policyRes.Allowed {
		return s.deny(base, "policy denied: "+policyRes.Reason, &policyRes, nil, nil, nil), nil
	}

	// 5. Permission
	permRes := s.permissionEval.Evaluate(ctx, req, p.Ref())
	base.Permission = &permRes
	if !permRes.Granted {
		return s.deny(base, "permission denied: "+permRes.Reason, &policyRes, &permRes, nil, nil), nil
	}

	// 6. Risk
	riskRes := s.riskEval.Evaluate(ctx, req)
	base.Risk = &riskRes
	if !riskRes.Accepted {
		return s.deny(base, "risk rejected: "+riskRes.Reason, &policyRes, &permRes, &riskRes, nil), nil
	}

	// 7. Approval
	apprRes := s.approvalEval.Evaluate(ctx, req, riskRes)
	base.Approval = &apprRes
	switch apprRes.Status {
	case contracts.ApprovalStatusDenied:
		return s.deny(base, "approval denied: "+apprRes.Reason, &policyRes, &permRes, &riskRes, &apprRes), nil
	case contracts.ApprovalStatusPending:
		return s.deny(base, "approval required: "+apprRes.Reason, &policyRes, &permRes, &riskRes, &apprRes), nil
	case contracts.ApprovalStatusExpired:
		return s.deny(base, "approval expired: "+apprRes.Reason, &policyRes, &permRes, &riskRes, &apprRes), nil
	case contracts.ApprovalStatusApproved, contracts.ApprovalStatusSkipped:
		// proceed
	default:
		return s.deny(base, "approval status unknown", &policyRes, &permRes, &riskRes, &apprRes), nil
	}

	// Allowed
	base.Outcome = contracts.DecisionAllowed
	base.Reason = "authorized"
	base.Policy = &policyRes
	base.Permission = &permRes
	base.Risk = &riskRes
	base.Approval = &apprRes
	return base, nil
}

func (s *Service) deny(
	base contracts.Decision,
	reason string,
	policy *contracts.PolicyResult,
	perm *contracts.PermissionResult,
	risk *contracts.RiskResult,
	appr *contracts.ApprovalResult,
) contracts.Decision {
	base.Outcome = contracts.DecisionDenied
	base.Reason = reason
	base.Policy = policy
	base.Permission = perm
	base.Risk = risk
	base.Approval = appr
	return base
}

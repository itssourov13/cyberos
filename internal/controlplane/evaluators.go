package controlplane

import (
	"context"

	"cyberos/internal/contracts"
	domainscope "cyberos/internal/domain/scope"
)

// PolicyEvaluator evaluates policy for a request.
type PolicyEvaluator interface {
	Evaluate(ctx context.Context, req EvaluateRequest, principal contracts.PrincipalRef, sc *domainscope.Scope) contracts.PolicyResult
}

// PermissionEvaluator evaluates permission for a request.
type PermissionEvaluator interface {
	Evaluate(ctx context.Context, req EvaluateRequest, principal contracts.PrincipalRef) contracts.PermissionResult
}

// RiskEvaluator evaluates risk for a request.
type RiskEvaluator interface {
	Evaluate(ctx context.Context, req EvaluateRequest) contracts.RiskResult
}

// ApprovalEvaluator evaluates whether approval is satisfied.
type ApprovalEvaluator interface {
	Evaluate(ctx context.Context, req EvaluateRequest, risk contracts.RiskResult) contracts.ApprovalResult
}

// DefaultPolicyEvaluator allows all requests that already passed scope/principal checks.
// Future phases will replace this with real policy documents.
type DefaultPolicyEvaluator struct{}

func (DefaultPolicyEvaluator) Evaluate(_ context.Context, req EvaluateRequest, _ contracts.PrincipalRef, _ *domainscope.Scope) contracts.PolicyResult {
	if req.CapabilityID == "" {
		return contracts.PolicyResult{Allowed: false, Reason: "capability is required"}
	}
	return contracts.PolicyResult{Allowed: true, Reason: "default policy allow"}
}

// DefaultPermissionEvaluator grants permission when a capability is present.
// Real permission binding arrives in later phases.
type DefaultPermissionEvaluator struct{}

func (DefaultPermissionEvaluator) Evaluate(_ context.Context, req EvaluateRequest, principal contracts.PrincipalRef) contracts.PermissionResult {
	if principal.ID == "" {
		return contracts.PermissionResult{Granted: false, Reason: "principal missing"}
	}
	if req.CapabilityID == "" {
		return contracts.PermissionResult{Granted: false, Reason: "capability missing"}
	}
	return contracts.PermissionResult{Granted: true, Reason: "default permission grant"}
}

// DefaultRiskEvaluator accepts low/medium/none; rejects critical unless hinted otherwise.
type DefaultRiskEvaluator struct{}

func (DefaultRiskEvaluator) Evaluate(_ context.Context, req EvaluateRequest) contracts.RiskResult {
	level := req.RiskLevelHint
	if level == "" {
		level = contracts.RiskLevelLow
	}
	if !level.Valid() {
		return contracts.RiskResult{Level: contracts.RiskLevelHigh, Accepted: false, Reason: "invalid risk level"}
	}
	if level == contracts.RiskLevelCritical {
		return contracts.RiskResult{Level: level, Accepted: false, Reason: "critical risk not accepted by default"}
	}
	return contracts.RiskResult{Level: level, Accepted: true, Reason: "risk accepted"}
}

// DefaultApprovalEvaluator skips approval for non-high risk; requires it when asked.
type DefaultApprovalEvaluator struct{}

func (DefaultApprovalEvaluator) Evaluate(_ context.Context, req EvaluateRequest, risk contracts.RiskResult) contracts.ApprovalResult {
	if req.RequireApproval {
		return contracts.ApprovalResult{
			Status: contracts.ApprovalStatusPending,
			Reason: "approval explicitly required",
		}
	}
	if risk.Level == contracts.RiskLevelHigh || risk.Level == contracts.RiskLevelCritical {
		return contracts.ApprovalResult{
			Status: contracts.ApprovalStatusPending,
			Reason: "high/critical risk requires approval",
		}
	}
	return contracts.ApprovalResult{
		Status: contracts.ApprovalStatusSkipped,
		Reason: "approval not required",
	}
}

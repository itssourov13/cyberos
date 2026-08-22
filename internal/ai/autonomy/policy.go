package autonomy

import (
	"fmt"
	"strings"
	"time"

	"cyberos/internal/contracts"
)

// AutonomyMode controls whether automatic workflow advancement is permitted.
type AutonomyMode string

const (
	ModeManual               AutonomyMode = "MANUAL"
	ModeAssisted             AutonomyMode = "ASSISTED"
	ModeControlledAutonomous AutonomyMode = "CONTROLLED_AUTONOMOUS"
)

// PolicyContractV1 is the autonomy policy contract version.
const PolicyContractV1 contracts.SchemaVersion = "1"

// PolicyDecision is the result of policy evaluation (not a Grant).
type PolicyDecision string

const (
	DecisionAllow PolicyDecision = "ALLOW"
	DecisionDeny  PolicyDecision = "DENY"
	DecisionWait  PolicyDecision = "WAIT" // MANUAL/ASSISTED — no auto advance
	DecisionStop  PolicyDecision = "STOP"
)

// AutonomyPolicy is an execution-eligibility gate above the security kernel.
// It never issues grants or calls Broker.
type AutonomyPolicy struct {
	Version             contracts.SchemaVersion  `json:"version"`
	Mode                AutonomyMode             `json:"mode"`
	ProjectID           contracts.ProjectID      `json:"project_id"`
	TenantID            contracts.TenantID       `json:"tenant_id,omitempty"`
	ScopeID             contracts.ScopeID        `json:"scope_id,omitempty"`
	AllowedCapabilities []contracts.CapabilityID `json:"allowed_capabilities,omitempty"`
	DeniedCapabilities  []contracts.CapabilityID `json:"denied_capabilities,omitempty"`
	MaxSteps            int                      `json:"max_steps,omitempty"`
	MaxRuntime          time.Duration            `json:"max_runtime,omitempty"`
	MaxOutputBytes      int64                    `json:"max_output_bytes,omitempty"`
	MaxConcurrentSteps  int                      `json:"max_concurrent_steps,omitempty"`
	MaxRisk             string                   `json:"max_risk,omitempty"` // low|medium|high|critical
	RequireRiskWhenAuto bool                     `json:"require_risk_when_auto,omitempty"`
}

// PolicyEvalRequest is the authoritative context for a policy check.
type PolicyEvalRequest struct {
	ProjectID       contracts.ProjectID
	TenantID        contracts.TenantID
	ScopeID         contracts.ScopeID
	PrincipalID     contracts.PrincipalID
	CorrelationID   contracts.CorrelationID
	CapabilityID    contracts.CapabilityID
	StepsExecuted   int
	RuntimeElapsed  time.Duration
	OutputBytes     int64
	ConcurrentSteps int
	RiskLevel       string // optional
	Policy          AutonomyPolicy
}

// PolicyEvalResult is a deterministic eligibility decision (not authorization).
type PolicyEvalResult struct {
	Decision   PolicyDecision `json:"decision"`
	Reason     string         `json:"reason"`
	Mode       AutonomyMode   `json:"mode"`
	PolicyVer  string         `json:"policy_version"`
	Capability string         `json:"capability,omitempty"`
	ProjectID  string         `json:"project_id"`
}

// ValidatePolicy checks policy structure fail-closed.
func ValidatePolicy(p AutonomyPolicy) error {
	v := p.Version
	if v == "" {
		v = PolicyContractV1
	}
	if v != PolicyContractV1 {
		return fmt.Errorf("%w: unsupported policy version %q", contracts.ErrInvalidArgument, p.Version)
	}
	switch p.Mode {
	case ModeManual, ModeAssisted, ModeControlledAutonomous:
	default:
		return fmt.Errorf("%w: unknown autonomy mode %q", contracts.ErrInvalidArgument, p.Mode)
	}
	if p.ProjectID == "" {
		return fmt.Errorf("%w: policy project_id required", contracts.ErrInvalidArgument)
	}
	for _, c := range append(p.AllowedCapabilities, p.DeniedCapabilities...) {
		s := string(c)
		if s == "" || looksLikeExecutable(s) || looksLikeShell(s) {
			return fmt.Errorf("%w: invalid capability in policy %q", contracts.ErrInvalidArgument, s)
		}
	}
	if p.MaxSteps < 0 || p.MaxOutputBytes < 0 || p.MaxConcurrentSteps < 0 {
		return fmt.Errorf("%w: negative policy budget", contracts.ErrInvalidArgument)
	}
	if p.MaxRisk != "" {
		switch strings.ToLower(p.MaxRisk) {
		case "low", "medium", "high", "critical":
		default:
			return fmt.Errorf("%w: invalid max_risk", contracts.ErrInvalidArgument)
		}
	}
	return nil
}

// EvaluateAutonomyPolicy decides ALLOW/DENY/WAIT/STOP. Never issues grants.
func EvaluateAutonomyPolicy(req PolicyEvalRequest) (PolicyEvalResult, error) {
	res := PolicyEvalResult{
		Mode: req.Policy.Mode, PolicyVer: string(req.Policy.Version),
		Capability: string(req.CapabilityID), ProjectID: string(req.ProjectID),
	}
	if err := ValidatePolicy(req.Policy); err != nil {
		res.Decision = DecisionDeny
		res.Reason = "malformed_policy"
		return res, err
	}
	if req.Policy.Version == "" {
		res.PolicyVer = string(PolicyContractV1)
	}
	// Identity binding
	if req.Policy.ProjectID != req.ProjectID {
		res.Decision = DecisionDeny
		res.Reason = "project_mismatch"
		return res, nil
	}
	if req.Policy.TenantID != "" && req.TenantID != "" && req.Policy.TenantID != req.TenantID {
		res.Decision = DecisionDeny
		res.Reason = "tenant_mismatch"
		return res, nil
	}
	if req.Policy.ScopeID != "" && req.ScopeID != "" && req.Policy.ScopeID != req.ScopeID {
		res.Decision = DecisionDeny
		res.Reason = "scope_mismatch"
		return res, nil
	}

	// Mode gate
	switch req.Policy.Mode {
	case ModeManual, ModeAssisted:
		res.Decision = DecisionWait
		res.Reason = "mode_" + strings.ToLower(string(req.Policy.Mode))
		return res, nil
	case ModeControlledAutonomous:
		// continue budget checks
	default:
		res.Decision = DecisionDeny
		res.Reason = "unknown_mode"
		return res, nil
	}

	// Capability deny > allow
	cap := req.CapabilityID
	if cap != "" {
		for _, d := range req.Policy.DeniedCapabilities {
			if d == cap {
				res.Decision = DecisionDeny
				res.Reason = "capability_denied"
				return res, nil
			}
		}
		if len(req.Policy.AllowedCapabilities) > 0 {
			ok := false
			for _, a := range req.Policy.AllowedCapabilities {
				if a == cap {
					ok = true
					break
				}
			}
			if !ok {
				res.Decision = DecisionDeny
				res.Reason = "capability_not_allowed"
				return res, nil
			}
		}
	}

	if req.Policy.MaxSteps > 0 && req.StepsExecuted >= req.Policy.MaxSteps {
		res.Decision = DecisionStop
		res.Reason = "max_steps_reached"
		return res, nil
	}
	if req.Policy.MaxRuntime > 0 && req.RuntimeElapsed >= req.Policy.MaxRuntime {
		res.Decision = DecisionStop
		res.Reason = "runtime_exceeded"
		return res, nil
	}
	if req.Policy.MaxOutputBytes > 0 && req.OutputBytes >= req.Policy.MaxOutputBytes {
		res.Decision = DecisionStop
		res.Reason = "output_exceeded"
		return res, nil
	}
	if req.Policy.MaxConcurrentSteps > 0 && req.ConcurrentSteps >= req.Policy.MaxConcurrentSteps {
		res.Decision = DecisionDeny
		res.Reason = "concurrency_exceeded"
		return res, nil
	}
	if req.Policy.RequireRiskWhenAuto && req.RiskLevel == "" {
		res.Decision = DecisionDeny
		res.Reason = "risk_required"
		return res, nil
	}
	if req.Policy.MaxRisk != "" && req.RiskLevel != "" {
		if riskRank(req.RiskLevel) > riskRank(req.Policy.MaxRisk) {
			res.Decision = DecisionDeny
			res.Reason = "risk_exceeded"
			return res, nil
		}
	}

	res.Decision = DecisionAllow
	res.Reason = "policy_allow"
	return res, nil
}

func riskRank(s string) int {
	switch strings.ToLower(s) {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	default:
		return 0
	}
}

// MayAutoAdvance reports whether CONTROLLED_AUTONOMOUS allow was granted by policy.
// Caller must still run CP→Grant via Orchestrator — this is not authorization.
func MayAutoAdvance(d PolicyEvalResult) bool {
	return d.Decision == DecisionAllow
}

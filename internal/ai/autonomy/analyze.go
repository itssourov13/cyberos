package autonomy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cyberos/internal/ai"
	"cyberos/internal/ai/aicontext"
	"cyberos/internal/contracts"
)

// Analyzer performs Observe → AIContext → Brain.Analyze → AnalysisResult.
// It never advances workflows, issues grants, or executes tools.
type Analyzer struct {
	Brain      ai.Brain
	Assembler  *aicontext.Assembler
	Limits     Limits
	SourceData aicontext.SourceData // optional pre-filtered observation inputs
	Now        func() time.Time
}

// NewAnalyzer constructs an analyzer. Brain defaults to MockBrain if nil.
func NewAnalyzer(brain ai.Brain, asm *aicontext.Assembler) *Analyzer {
	if brain == nil {
		brain = &ai.MockBrain{}
	}
	if asm == nil {
		asm = aicontext.NewAssembler()
	}
	return &Analyzer{Brain: brain, Assembler: asm, Limits: DefaultLimits(), Now: time.Now}
}

// AnalyzeObservations builds context from SourceData under request isolation,
// calls Brain.Analyze, and returns a validated untrusted AnalysisResult.
func (a *Analyzer) AnalyzeObservations(ctx context.Context, req ObservationRequest) (*AnalysisResult, error) {
	if a == nil {
		return nil, fmt.Errorf("%w: analyzer not configured", contracts.ErrInvalidArgument)
	}
	if req.ProjectID == "" {
		return nil, fmt.Errorf("%w: project_id required", contracts.ErrInvalidArgument)
	}
	if strings.TrimSpace(req.Objective) == "" {
		return nil, fmt.Errorf("%w: objective required", contracts.ErrInvalidArgument)
	}

	creq := aicontext.ContextRequest{
		TenantID: req.TenantID, ProjectID: req.ProjectID, ScopeID: req.ScopeID,
		PrincipalID: req.PrincipalID, CorrelationID: req.CorrelationID,
		IncludeFindings: true, IncludeEvidence: true, IncludeCapabilities: true,
	}
	aiCtx, err := a.Assembler.Assemble(creq, a.SourceData)
	if err != nil {
		return nil, err
	}
	// Authoritative isolation already applied by assembler.

	brainReq := ai.BrainRequest{
		Operation: ai.OpAnalyze, PrincipalID: req.PrincipalID, ProjectID: req.ProjectID,
		ScopeID: req.ScopeID, TenantID: req.TenantID, Objective: req.Objective,
		CorrelationID: req.CorrelationID,
	}
	resp, err := a.Brain.Analyze(ctx, brainReq)
	if err != nil {
		return nil, err
	}
	if resp.Status == "error" {
		return nil, fmt.Errorf("%w: brain analysis failed: %s", contracts.ErrInvalidArgument, resp.Error)
	}

	now := time.Now()
	if a.Now != nil {
		now = a.Now()
	}
	summary := ""
	if resp.Analysis != nil {
		summary = resp.Analysis.Summary
	}
	result := &AnalysisResult{
		ContractVersion: AnalysisContractV1,
		AnalysisID:      "an-" + string(req.CorrelationID),
		ProjectID:       req.ProjectID, // authoritative — ignore model
		TenantID:        req.TenantID,
		ScopeID:         req.ScopeID,
		PrincipalID:     req.PrincipalID,
		CorrelationID:   req.CorrelationID,
		CausationID:     req.CausationID,
		CreatedAt:       now,
		Summary:         summary,
		Trust:           TrustUntrusted,
		ProviderID:      req.ProviderID,
		Model:           req.Model,
		SourceRef:       req.SourceRef,
	}
	if result.AnalysisID == "an-" {
		result.AnalysisID = fmt.Sprintf("an-%d", now.UnixNano())
	}
	if resp.ProviderMeta != nil {
		if result.ProviderID == "" {
			result.ProviderID = resp.ProviderMeta["provider"]
		}
		if result.Model == "" {
			result.Model = resp.ProviderMeta["model"]
		}
	}
	// Optional: parse recommended capabilities from analysis details (capability IDs only)
	if resp.Analysis != nil && resp.Analysis.Details != nil {
		if caps, ok := resp.Analysis.Details["recommended_capabilities"]; ok {
			for _, part := range strings.Split(caps, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				result.RecommendedCapabilities = append(result.RecommendedCapabilities, contracts.CapabilityID(part))
			}
		}
	}
	_ = aiCtx // context was assembled for future provider adapters; MockBrain ignores it
	if err := ValidateAnalysisResult(result, a.Limits); err != nil {
		return nil, err
	}
	return result, nil
}

// ValidateAnalysisResult fail-closed structural validation.
func ValidateAnalysisResult(r *AnalysisResult, lim Limits) error {
	if r == nil {
		return fmt.Errorf("%w: analysis result nil", contracts.ErrInvalidArgument)
	}
	if lim.MaxSummaryLen == 0 {
		lim = DefaultLimits()
	}
	cv := r.ContractVersion
	if cv == "" {
		cv = AnalysisContractV1
	}
	if cv != AnalysisContractV1 {
		return fmt.Errorf("%w: unsupported analysis contract %q", contracts.ErrInvalidArgument, r.ContractVersion)
	}
	if r.ProjectID == "" {
		return fmt.Errorf("%w: project_id required", contracts.ErrInvalidArgument)
	}
	if r.Trust != TrustUntrusted {
		r.Trust = TrustUntrusted
	}
	if len(r.Summary) > lim.MaxSummaryLen {
		return fmt.Errorf("%w: summary too long", contracts.ErrInvalidArgument)
	}
	if len(r.Items) > lim.MaxItems {
		return fmt.Errorf("%w: too many analysis items", contracts.ErrInvalidArgument)
	}
	for i, it := range r.Items {
		if len(it.Summary) > lim.MaxItemLen {
			return fmt.Errorf("%w: item %d too long", contracts.ErrInvalidArgument, i)
		}
	}
	if len(r.RecommendedCapabilities) > lim.MaxCapabilities {
		return fmt.Errorf("%w: too many recommended capabilities", contracts.ErrInvalidArgument)
	}
	for _, c := range r.RecommendedCapabilities {
		s := string(c)
		if s == "" {
			return fmt.Errorf("%w: empty capability recommendation", contracts.ErrInvalidArgument)
		}
		if looksLikeExecutable(s) || looksLikeShell(s) {
			return fmt.Errorf("%w: recommended capability must not be executable/shell %q", contracts.ErrInvalidArgument, s)
		}
	}
	return nil
}

func looksLikeExecutable(s string) bool {
	if strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") {
		return true
	}
	switch strings.ToLower(s) {
	case "nmap", "curl", "bash", "sh", "cmd":
		return true
	}
	return false
}

func looksLikeShell(s string) bool {
	return strings.Contains(s, "&&") || strings.Contains(s, ";") || strings.Contains(s, "sh -c") ||
		strings.Contains(s, " ") && (strings.HasPrefix(s, "nmap ") || strings.HasPrefix(s, "curl "))
}

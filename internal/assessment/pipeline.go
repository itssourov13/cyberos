package assessment

import (
	"context"
	"fmt"

	"cyberos/internal/assets"
	"cyberos/internal/cases"
	"cyberos/internal/contracts"
	"cyberos/internal/findings"
	"cyberos/internal/riskprio"
	"cyberos/internal/tools"
)

// Phase6Sink is the optional post-execution product pipeline (Phase 6H).
// It never authorizes or calls the Broker.
type Phase6Sink struct {
	Assets   *assets.Inventory
	Findings *findings.Service
	Risk     *riskprio.Engine
	Cases    *cases.Service
}

// AfterSuccessfulStep maps execution output into assets → findings → risk → case.
func (p *Phase6Sink) AfterSuccessfulStep(
	_ context.Context,
	a *Assessment,
	step contracts.AssessmentStep,
	grant contracts.ExecutionGrant,
	res contracts.ExecutionResult,
) {
	if p == nil || a == nil {
		return
	}
	nr := normalizedFromResult(res, step)
	var assetID contracts.AssetID
	if p.Assets != nil {
		as, _, err := p.Assets.IngestFromResult(a.ProjectID, a.ScopeID, grant.TaskID, string(grant.CorrelationID)+"/"+step.ID, nr)
		if err == nil && as != nil {
			assetID = as.ID
		}
	}
	var findingID contracts.FindingID
	if p.Findings != nil {
		ingested, err := p.Findings.Ingest(findings.IngestRequest{
			Result: nr, ProjectID: a.ProjectID, ScopeID: a.ScopeID, TaskID: grant.TaskID,
			AssessmentID: string(a.ID), ExecutionID: string(grant.CorrelationID) + "/" + step.ID, PrincipalID: grant.Principal.ID,
			CorrelationID:   grant.CorrelationID,
			EvidenceIDs:     nil,
			DefaultSeverity: contracts.FindingSeverityInfo,
		})
		if err == nil && len(ingested) > 0 && ingested[0].Finding != nil {
			findingID = ingested[0].Finding.ID
			if p.Risk != nil {
				_, _ = p.Risk.EvaluateFinding(ingested[0].Finding, contracts.AssetCriticalityMedium, contracts.ExposureUnknown, assetID)
			}
		}
	}
	if p.Cases != nil && findingID != "" {
		caseID := fmt.Sprintf("case-%s", a.ID)
		if _, err := p.Cases.Get(caseID); err != nil {
			_, _ = p.Cases.Create(cases.CreateRequest{
				ID: caseID, ProjectID: a.ProjectID, AssessmentID: string(a.ID),
				Title: "Assessment case: " + a.Title, FindingIDs: []contracts.FindingID{findingID},
				AssetIDs: func() []contracts.AssetID {
					if assetID != "" {
						return []contracts.AssetID{assetID}
					}
					return nil
				}(),
				CreatedBy: grant.Principal.ID,
			})
		} else {
			_, _ = p.Cases.LinkFinding(caseID, findingID)
			if assetID != "" {
				_, _ = p.Cases.LinkAsset(caseID, assetID)
			}
		}
		if p.Risk != nil {
			_, _ = p.Cases.DerivePriorityFromRisks(caseID, p.Risk)
		}
	}
}

func normalizedFromResult(res contracts.ExecutionResult, step contracts.AssessmentStep) tools.NormalizedResult {
	nr := tools.NormalizedResult{
		ToolID:     res.ExecutorID,
		Capability: string(step.CapabilityID),
		Success:    res.Status == contracts.ExecutionStatusSucceeded,
		Summary:    res.ErrorMessage,
		Raw:        map[string]any{},
	}
	if nr.Summary == "" {
		if s, ok := res.Output["summary"].(string); ok {
			nr.Summary = s
		} else {
			nr.Summary = "assessment step " + step.ID
		}
	}
	if res.Output != nil {
		for k, v := range res.Output {
			nr.Raw[k] = v
		}
	}
	if t, ok := res.Output["target"].(string); ok {
		nr.Raw["target"] = t
	}
	nr.Findings = []tools.FindingHint{{
		Title: nr.Summary, Severity: "INFO", Detail: step.ID,
	}}
	return nr
}

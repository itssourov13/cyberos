package assessment

import (
	"context"
	"errors"
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
	ctx context.Context,
	a *Assessment,
	step contracts.AssessmentStep,
	grant contracts.ExecutionGrant,
	res contracts.ExecutionResult,
) error {
	if p == nil || a == nil {
		return nil
	}
	nr := normalizedFromResult(res, step)
	var assetID contracts.AssetID
	if p.Assets != nil {
		as, _, err := p.Assets.IngestFromResult(a.ProjectID, a.ScopeID, grant.TaskID, string(grant.CorrelationID)+"/"+step.ID, nr)
		if err != nil {
			return fmt.Errorf("persist assessment asset observations: %w", err)
		}
		if as != nil {
			assetID = as.ID
		}
	}
	var findingID contracts.FindingID
	if p.Findings != nil {
		ingested, err := p.Findings.Ingest(findings.IngestRequest{
			Result: nr, ProjectID: a.ProjectID, ScopeID: a.ScopeID, TaskID: grant.TaskID,
			AssessmentID: string(a.ID), ExecutionID: string(grant.CorrelationID) + "/" + step.ID, PrincipalID: grant.Principal.ID,
			CorrelationID: grant.CorrelationID, DefaultSeverity: contracts.FindingSeverityInfo,
		})
		if err != nil {
			return fmt.Errorf("persist assessment findings: %w", err)
		}
		if len(ingested) > 0 && ingested[0].Finding != nil {
			findingID = ingested[0].Finding.ID
			if p.Risk != nil {
				if _, err := p.Risk.EvaluateFinding(ingested[0].Finding, contracts.AssetCriticalityMedium, contracts.ExposureUnknown, assetID); err != nil {
					return fmt.Errorf("persist assessment risk evaluation: %w", err)
				}
			}
		}
	}
	if p.Cases != nil && findingID != "" {
		caseID := fmt.Sprintf("case-%s", a.ID)
		_, err := p.Cases.Get(caseID)
		switch {
		case errors.Is(err, contracts.ErrNotFound):
			assets := []contracts.AssetID(nil)
			if assetID != "" {
				assets = []contracts.AssetID{assetID}
			}
			_, err = p.Cases.Create(cases.CreateRequest{
				ID: caseID, ProjectID: a.ProjectID, AssessmentID: string(a.ID),
				Title: "Assessment case: " + a.Title, FindingIDs: []contracts.FindingID{findingID},
				AssetIDs: assets, CreatedBy: grant.Principal.ID,
			})
			if err != nil {
				return fmt.Errorf("create assessment case: %w", err)
			}
		case err != nil:
			return fmt.Errorf("load assessment case: %w", err)
		default:
			if _, err := p.Cases.LinkFinding(caseID, findingID); err != nil {
				return fmt.Errorf("link finding to assessment case: %w", err)
			}
			if assetID != "" {
				if _, err := p.Cases.LinkAsset(caseID, assetID); err != nil {
					return fmt.Errorf("link asset to assessment case: %w", err)
				}
			}
		}
		if p.Risk != nil {
			if _, err := p.Cases.DerivePriorityFromRisks(caseID, p.Risk); err != nil {
				return fmt.Errorf("derive assessment case priority: %w", err)
			}
		}
	}
	return nil
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

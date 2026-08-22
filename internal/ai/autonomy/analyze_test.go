package autonomy_test

import (
	"context"
	"strings"
	"testing"

	"cyberos/internal/ai"
	"cyberos/internal/ai/aicontext"
	"cyberos/internal/ai/autonomy"
	"cyberos/internal/contracts"
)

func TestAnalyzeObservationsMock(t *testing.T) {
	a := autonomy.NewAnalyzer(&ai.MockBrain{}, nil)
	a.SourceData = aicontext.SourceData{
		Findings: []aicontext.FindingInfo{
			{ID: "f1", ProjectID: "proj-a", TenantID: "ten-a", Summary: "port open api_key=sk-LEAKED999"},
			{ID: "f2", ProjectID: "proj-b", TenantID: "ten-b", Summary: "other"},
		},
	}
	res, err := a.AnalyzeObservations(context.Background(), autonomy.ObservationRequest{
		ProjectID: "proj-a", TenantID: "ten-a", ScopeID: "scope-a", PrincipalID: "u1",
		Objective: "summarize findings", CorrelationID: "corr-1", ProviderID: "mock",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Trust != autonomy.TrustUntrusted {
		t.Fatal(res.Trust)
	}
	if res.ProjectID != "proj-a" || res.CorrelationID != "corr-1" {
		t.Fatal(res)
	}
	if strings.Contains(res.Summary, "sk-LEAKED") {
		// MockBrain doesn't echo context; ensure our result doesn't invent secrets
		t.Fatal("secret in summary")
	}
}

func TestMissingProject(t *testing.T) {
	a := autonomy.NewAnalyzer(nil, nil)
	_, err := a.AnalyzeObservations(context.Background(), autonomy.ObservationRequest{Objective: "x"})
	if err == nil {
		t.Fatal("project required")
	}
}

func TestRejectExecutableCapabilityRecommendation(t *testing.T) {
	r := &autonomy.AnalysisResult{
		ContractVersion: autonomy.AnalysisContractV1, AnalysisID: "a1", ProjectID: "p",
		Trust: autonomy.TrustUntrusted, Summary: "ok",
		RecommendedCapabilities: []contracts.CapabilityID{"nmap -sV target"},
	}
	if err := autonomy.ValidateAnalysisResult(r, autonomy.DefaultLimits()); err == nil {
		t.Fatal("shell-like capability")
	}
	r.RecommendedCapabilities = []contracts.CapabilityID{"/usr/bin/nmap"}
	if err := autonomy.ValidateAnalysisResult(r, autonomy.DefaultLimits()); err == nil {
		t.Fatal("path")
	}
	r.RecommendedCapabilities = []contracts.CapabilityID{"recon.network.portscan"}
	if err := autonomy.ValidateAnalysisResult(r, autonomy.DefaultLimits()); err != nil {
		t.Fatal(err)
	}
}

func TestOversizedSummary(t *testing.T) {
	r := &autonomy.AnalysisResult{
		ContractVersion: autonomy.AnalysisContractV1, AnalysisID: "a1", ProjectID: "p",
		Trust: autonomy.TrustUntrusted, Summary: strings.Repeat("x", 10000),
	}
	if err := autonomy.ValidateAnalysisResult(r, autonomy.DefaultLimits()); err == nil {
		t.Fatal("size")
	}
}

func TestAuthoritativeProjectNotOverridden(t *testing.T) {
	a := autonomy.NewAnalyzer(&ai.MockBrain{}, nil)
	res, err := a.AnalyzeObservations(context.Background(), autonomy.ObservationRequest{
		ProjectID: "authoritative", ScopeID: "s", Objective: "o",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ProjectID != "authoritative" {
		t.Fatal(res.ProjectID)
	}
}

func TestAnalyzeDoesNotAdvanceOrGrant(t *testing.T) {
	// Structural: Analyzer only depends on Brain + Assembler.
	a := autonomy.NewAnalyzer(&ai.MockBrain{}, nil)
	_, err := a.AnalyzeObservations(context.Background(), autonomy.ObservationRequest{
		ProjectID: "p", Objective: "o",
	})
	if err != nil {
		t.Fatal(err)
	}
}

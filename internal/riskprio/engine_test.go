package riskprio

import (
	"testing"

	"cyberos/internal/contracts"
	domainfinding "cyberos/internal/domain/finding"
)

func TestScoreDeterministic(t *testing.T) {
	in := Input{
		FindingID: "f1", ProjectID: "p", Severity: contracts.FindingSeverityHigh,
		Confidence:       contracts.FindingConfidenceHigh,
		AssetCriticality: contracts.AssetCriticalityCritical,
		Exposure:         contracts.ExposureInternetFacing,
	}
	s1, p1, f1 := Score(in)
	s2, p2, _ := Score(in)
	if s1 != s2 || p1 != p2 {
		t.Fatal(s1, s2, p1, p2)
	}
	if s1 < 85 || p1 != contracts.RiskPriorityCritical {
		t.Fatalf("score=%d priority=%s factors=%v", s1, p1, f1)
	}
}

func TestMediumFindingCriticalAsset(t *testing.T) {
	in := Input{
		Severity: contracts.FindingSeverityMedium, Confidence: contracts.FindingConfidenceMedium,
		AssetCriticality: contracts.AssetCriticalityCritical, Exposure: contracts.ExposureInternetFacing,
	}
	score, prio, _ := Score(in)
	if prio == contracts.RiskPriorityInfo {
		t.Fatalf("expected elevated priority, got %s score=%d", prio, score)
	}
}

func TestEvaluateAndSummary(t *testing.T) {
	e := NewEngine()
	f, _ := domainfinding.New("f1", "t1", "p1", "s1", contracts.FindingSeverityHigh, "Title")
	f.AssessmentID = "a1"
	f.Confidence = contracts.FindingConfidenceHigh
	ev, err := e.EvaluateFinding(f, contracts.AssetCriticalityHigh, contracts.ExposureExternal, "asset-1")
	if err != nil || ev.ModelVersion != ModelVersion {
		t.Fatal(err, ev)
	}
	sum := e.SummarizeAssessment("a1")
	if sum.Total != 1 {
		t.Fatal(sum)
	}
	// recalculate
	ev2, _ := e.EvaluateFinding(f, contracts.AssetCriticalityCritical, contracts.ExposureInternetFacing, "asset-1")
	if ev2.Score <= ev.Score {
		t.Fatalf("recalc should increase score: %d vs %d", ev2.Score, ev.Score)
	}
}

package cases

import (
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/riskprio"
)

func TestCaseLifecycle(t *testing.T) {
	s := NewService()
	c, err := s.Create(CreateRequest{
		ID: "c1", ProjectID: "p1", AssessmentID: "a1", Title: "Case",
		FindingIDs: []contracts.FindingID{"f1"}, AssetIDs: []contracts.AssetID{"asset-1"},
	})
	if err != nil || c.Status != contracts.CaseStatusOpen {
		t.Fatal(err, c)
	}
	if _, err := s.TransitionStatus("c1", contracts.CaseStatusAcknowledged); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TransitionStatus("c1", contracts.CaseStatusInProgress); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TransitionStatus("c1", contracts.CaseStatusResolved); err != nil {
		t.Fatal(err)
	}
	if _, err := s.TransitionStatus("c1", contracts.CaseStatusOpen); err == nil {
		t.Fatal("invalid transition")
	}
	if _, err := s.TransitionStatus("c1", contracts.CaseStatusReopened); err != nil {
		t.Fatal(err)
	}
}

func TestDerivePriority(t *testing.T) {
	s := NewService()
	_, _ = s.Create(CreateRequest{ID: "c2", ProjectID: "p", AssessmentID: "a", Title: "T", FindingIDs: []contracts.FindingID{"f9"}})
	eng := riskprio.NewEngine()
	_, _ = eng.Evaluate(riskprio.Input{
		FindingID: "f9", ProjectID: "p", Severity: contracts.FindingSeverityCritical,
		Confidence: contracts.FindingConfidenceHigh, AssetCriticality: contracts.AssetCriticalityCritical,
		Exposure: contracts.ExposureInternetFacing,
	})
	c, err := s.DerivePriorityFromRisks("c2", eng)
	if err != nil || c.Priority != contracts.RiskPriorityCritical {
		t.Fatalf("%v %+v", err, c)
	}
}

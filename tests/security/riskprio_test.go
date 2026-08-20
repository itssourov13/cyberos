package security_test

import (
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/riskprio"
)

func TestSecurity_RiskDoesNotAuthorize(t *testing.T) {
	e := riskprio.NewEngine()
	ev, err := e.Evaluate(riskprio.Input{
		FindingID: "f1", ProjectID: "p",
		Severity:         contracts.FindingSeverityCritical,
		Confidence:       contracts.FindingConfidenceHigh,
		AssetCriticality: contracts.AssetCriticalityCritical,
		Exposure:         contracts.ExposureInternetFacing,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ev.Priority != contracts.RiskPriorityCritical {
		t.Fatal(ev.Priority)
	}
	// Critical risk still does not issue grants or execute tools.
}

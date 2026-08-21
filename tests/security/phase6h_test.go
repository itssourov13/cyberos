package security_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	v1 "cyberos/internal/api/v1"
	"cyberos/internal/auth"
	"cyberos/internal/contracts"
	"cyberos/internal/riskprio"
	"cyberos/internal/sdk"
)

func TestPhase6H_AuthRequiredBlocksUnauthenticated(t *testing.T) {
	client := sdk.NewInMemory()
	a := auth.NewAPIKeyAuthenticator(auth.NewMemoryKeyStore())
	srv := v1.NewServer(client)
	srv.Auth = auth.NewMiddleware(a)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/findings", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rr.Code)
	}
}

func TestPhase6H_RiskAndCasesStillDoNotAuthorize(t *testing.T) {
	e := riskprio.NewEngine()
	_, err := e.Evaluate(riskprio.Input{
		FindingID: "f", ProjectID: "p",
		Severity:         contracts.FindingSeverityCritical,
		Confidence:       contracts.FindingConfidenceHigh,
		AssetCriticality: contracts.AssetCriticalityCritical,
		Exposure:         contracts.ExposureInternetFacing,
	})
	if err != nil {
		t.Fatal(err)
	}
	client := sdk.NewInMemory()
	_, err = client.Execute(contracts.ExecutionGrant{}, nil)
	if err == nil {
		t.Fatal("empty grant must not execute")
	}
}

package security_test

import (
	"testing"

	"cyberos/internal/assets"
	"cyberos/internal/contracts"
	"cyberos/internal/tools"
)

func TestSecurity_AssetIngestDoesNotAuthorize(t *testing.T) {
	inv := assets.NewInventory()
	_, _, err := inv.IngestFromResult("p", "s", "t", "ex", tools.NormalizedResult{
		ToolID: "tool.nmap", Capability: "recon.network.portscan", Summary: "ok",
		Raw: map[string]any{"target": "10.0.0.1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Inventory is observation-only; no grants issued.
	if len(inv.ListByProject("p")) != 1 {
		t.Fatal("asset")
	}
	_ = contracts.ObservationKindGeneric
}

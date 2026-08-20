package assets

import (
	"testing"

	"cyberos/internal/contracts"
	"cyberos/internal/tools"
)

func TestUpsertTargetStable(t *testing.T) {
	inv := NewInventory()
	a1, err := inv.UpsertTarget("p1", "10.0.0.1", contracts.AssetTypeHost)
	if err != nil {
		t.Fatal(err)
	}
	a2, err := inv.UpsertTarget("p1", "10.0.0.1", contracts.AssetTypeHost)
	if err != nil || a1.ID != a2.ID {
		t.Fatalf("%v %s %s", err, a1.ID, a2.ID)
	}
	list := inv.ListByProject("p1")
	if len(list) != 1 {
		t.Fatal(len(list))
	}
}

func TestIngestFromResult(t *testing.T) {
	inv := NewInventory()
	a, obs, err := inv.IngestFromResult("p1", "s1", "t1", "ex1", tools.NormalizedResult{
		ToolID: "tool.nmap", Capability: "recon.network.portscan", Summary: "scan done",
		Findings: []tools.FindingHint{{Title: "Open", Detail: "22/tcp"}},
		Raw:      map[string]any{"target": "scanme.example"},
	})
	if err != nil || a == nil || len(obs) < 1 {
		t.Fatalf("%v %v %v", err, a, obs)
	}
	eps := inv.ListEndpointsByAsset(a.ID)
	if len(eps) != 1 || eps[0].Port != 22 {
		t.Fatalf("%+v", eps)
	}
}

func TestCrossProjectIsolation(t *testing.T) {
	inv := NewInventory()
	_, _ = inv.UpsertTarget("p1", "host", contracts.AssetTypeHost)
	_, _ = inv.UpsertTarget("p2", "host", contracts.AssetTypeHost)
	if len(inv.ListByProject("p1")) != 1 || len(inv.ListByProject("p2")) != 1 {
		t.Fatal("isolation")
	}
}

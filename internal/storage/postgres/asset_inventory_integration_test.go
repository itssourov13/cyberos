//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"cyberos/internal/assets"
	"cyberos/internal/contracts"
	domainasset "cyberos/internal/domain/asset"
	"cyberos/internal/storage/postgres"
)

func TestPhase8B_AssetInventoryDurability(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()
	ar := postgres.NewAssetRepo(db)
	er := postgres.NewAssetEndpointRepo(db)
	or := postgres.NewAssetObservationRepo(db)
	inv := assets.NewInventory().WithStore(ar, er, or)

	a, err := inv.UpsertTarget("proj-8b", "192.168.1.10", contracts.AssetTypeHost)
	if err != nil {
		t.Fatal(err)
	}
	ep, err := inv.UpsertEndpoint(&domainasset.ServiceEndpoint{
		AssetID: a.ID, ProjectID: "proj-8b", Protocol: "tcp", Port: 443, Banner: "https",
	})
	if err != nil {
		t.Fatal(err)
	}
	o, err := domainasset.NewObservation("obs-8b", a.ID, "proj-8b", contracts.ObservationKindGeneric, "https open")
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.RecordObservation(o); err != nil {
		t.Fatal(err)
	}

	_ = db.Close()
	db2 := setupDB(t)
	inv2 := assets.NewInventory().WithStore(
		postgres.NewAssetRepo(db2),
		postgres.NewAssetEndpointRepo(db2),
		postgres.NewAssetObservationRepo(db2),
	)
	gotA, err := inv2.Get(a.ID)
	if err != nil || gotA.Value != "192.168.1.10" {
		t.Fatal(err, gotA)
	}
	gotEp, err := inv2.GetEndpoint(ep.ID)
	if err != nil || gotEp.Port != 443 {
		t.Fatal(err, gotEp)
	}
	gotO, err := inv2.GetObservation("obs-8b")
	if err != nil || gotO.Summary != "https open" {
		t.Fatal(err, gotO)
	}
	// project scoping on list
	eps, err := postgres.NewAssetEndpointRepo(db2).ListByProject(ctx, "other-proj", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range eps {
		if e.ID == ep.ID {
			t.Fatal("cross-project endpoint leak")
		}
	}
}

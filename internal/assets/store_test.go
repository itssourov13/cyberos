package assets

import (
	"context"
	"errors"
	"testing"

	"cyberos/internal/contracts"
	domainasset "cyberos/internal/domain/asset"
)

type memAssetR struct {
	m    map[contracts.AssetID]*domainasset.Asset
	fail bool
}

func (m *memAssetR) Create(ctx context.Context, a *domainasset.Asset) error { return m.Upsert(ctx, a) }
func (m *memAssetR) Upsert(_ context.Context, a *domainasset.Asset) error {
	if m.fail {
		return errors.New("db down")
	}
	if m.m == nil {
		m.m = map[contracts.AssetID]*domainasset.Asset{}
	}
	cp := *a
	m.m[a.ID] = &cp
	return nil
}
func (m *memAssetR) GetByID(_ context.Context, id contracts.AssetID) (*domainasset.Asset, error) {
	a, ok := m.m[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *a
	return &cp, nil
}
func (m *memAssetR) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainasset.Asset, error) {
	return nil, nil
}

type memEpR struct {
	m    map[string]*domainasset.ServiceEndpoint
	fail bool
}

func (m *memEpR) Upsert(_ context.Context, ep *domainasset.ServiceEndpoint) error {
	if m.fail {
		return errors.New("db down")
	}
	if m.m == nil {
		m.m = map[string]*domainasset.ServiceEndpoint{}
	}
	cp := *ep
	m.m[ep.ID] = &cp
	return nil
}
func (m *memEpR) GetByID(_ context.Context, id string) (*domainasset.ServiceEndpoint, error) {
	ep, ok := m.m[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *ep
	return &cp, nil
}
func (m *memEpR) ListByAsset(context.Context, contracts.AssetID, int, int) ([]*domainasset.ServiceEndpoint, error) {
	return nil, nil
}
func (m *memEpR) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainasset.ServiceEndpoint, error) {
	return nil, nil
}

type memObsR struct {
	m    map[string]*domainasset.Observation
	fail bool
}

func (m *memObsR) Upsert(_ context.Context, o *domainasset.Observation) error {
	if m.fail {
		return errors.New("db down")
	}
	if m.m == nil {
		m.m = map[string]*domainasset.Observation{}
	}
	cp := *o
	m.m[o.ID] = &cp
	return nil
}
func (m *memObsR) GetByID(_ context.Context, id string) (*domainasset.Observation, error) {
	o, ok := m.m[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *o
	return &cp, nil
}
func (m *memObsR) ListByAsset(context.Context, contracts.AssetID, int, int) ([]*domainasset.Observation, error) {
	return nil, nil
}
func (m *memObsR) ListByProject(context.Context, contracts.ProjectID, int, int) ([]*domainasset.Observation, error) {
	return nil, nil
}

func TestInventory_WithStoreReadThroughAndFail(t *testing.T) {
	ar, er, or := &memAssetR{}, &memEpR{}, &memObsR{}
	inv := NewInventory().WithStore(ar, er, or)
	a, err := inv.UpsertTarget("proj-b", "10.0.0.1", contracts.AssetTypeHost)
	if err != nil {
		t.Fatal(err)
	}
	inv.assets = map[contracts.AssetID]*domainasset.Asset{}
	inv.byValue = map[string]contracts.AssetID{}
	got, err := inv.Get(a.ID)
	if err != nil || got.Value != "10.0.0.1" {
		t.Fatal(err, got)
	}

	ep, err := inv.UpsertEndpoint(&domainasset.ServiceEndpoint{
		AssetID: a.ID, ProjectID: "proj-b", Protocol: "tcp", Port: 22,
	})
	if err != nil {
		t.Fatal(err)
	}
	inv.endpoints = map[string]*domainasset.ServiceEndpoint{}
	gotEp, err := inv.GetEndpoint(ep.ID)
	if err != nil || gotEp.Port != 22 {
		t.Fatal(err, gotEp)
	}

	o, err := domainasset.NewObservation("obs-1", a.ID, "proj-b", contracts.ObservationKindGeneric, "open")
	if err != nil {
		t.Fatal(err)
	}
	if err := inv.RecordObservation(o); err != nil {
		t.Fatal(err)
	}
	inv.obs = map[string]*domainasset.Observation{}
	gotO, err := inv.GetObservation("obs-1")
	if err != nil || gotO.Summary != "open" {
		t.Fatal(err, gotO)
	}

	failInv := NewInventory().WithStore(&memAssetR{fail: true}, er, or)
	if _, err := failInv.UpsertTarget("proj-b", "fail.example", contracts.AssetTypeDomain); err == nil {
		t.Fatal("expected persist error")
	}
}

// Package assets provides the Asset & Target Intelligence inventory.
// Observations feed findings; inventory never authorizes execution.
package assets

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cyberos/internal/contracts"
	domainasset "cyberos/internal/domain/asset"
	"cyberos/internal/storage/ports"
	"cyberos/internal/tools"
)

// Inventory is the asset/target intelligence store (memory + optional durable ports).
type Inventory struct {
	mu         sync.RWMutex
	assets     map[contracts.AssetID]*domainasset.Asset
	byValue    map[string]contracts.AssetID // project|value → asset id
	endpoints  map[string]*domainasset.ServiceEndpoint
	obs        map[string]*domainasset.Observation
	assetsR    ports.AssetRepository
	endpointsR ports.AssetEndpointRepository
	obsR       ports.AssetObservationRepository
}

// NewInventory constructs an empty inventory.
func NewInventory() *Inventory {
	return &Inventory{
		assets:    map[contracts.AssetID]*domainasset.Asset{},
		byValue:   map[string]contracts.AssetID{},
		endpoints: map[string]*domainasset.ServiceEndpoint{},
		obs:       map[string]*domainasset.Observation{},
	}
}

// WithStore attaches optional durable repositories (Phase 8B). Persist errors propagate.
func (inv *Inventory) WithStore(assets ports.AssetRepository, endpoints ports.AssetEndpointRepository, obs ports.AssetObservationRepository) *Inventory {
	if inv != nil {
		inv.assetsR = assets
		inv.endpointsR = endpoints
		inv.obsR = obs
	}
	return inv
}

// UpsertTarget creates or returns an asset for a target string within a project.
func (inv *Inventory) UpsertTarget(projectID contracts.ProjectID, target string, typ contracts.AssetType) (*domainasset.Asset, error) {
	target = strings.TrimSpace(target)
	if projectID == "" || target == "" {
		return nil, fmt.Errorf("%w: project and target required", contracts.ErrInvalidArgument)
	}
	if !typ.Valid() {
		typ = inferType(target)
	}
	key := string(projectID) + "|" + strings.ToLower(target)
	id := contracts.AssetID("asset-" + shortHash(key))
	if inv.assetsR != nil {
		// The durable repository is authoritative across API replicas. Resolve
		// the deterministic asset ID before creating, avoiding stale local value
		// indexes and duplicate assets after another replica has ingested it.
		existing, err := inv.assetsR.GetByID(context.Background(), id)
		if err == nil {
			existing.UpdatedAt = time.Now().UTC()
			if err := inv.assetsR.Upsert(context.Background(), existing); err != nil {
				return nil, fmt.Errorf("update existing asset: %w", err)
			}
			return existing, nil
		}
		if !errors.Is(err, contracts.ErrNotFound) {
			return nil, fmt.Errorf("resolve existing asset: %w", err)
		}
		a, err := domainasset.New(id, projectID, typ, target, target)
		if err != nil {
			return nil, err
		}
		if err := inv.assetsR.Upsert(context.Background(), a); err != nil {
			return nil, fmt.Errorf("persist new asset: %w", err)
		}
		return a, nil
	}
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if existingID, ok := inv.byValue[key]; ok {
		a := inv.assets[existingID]
		a.UpdatedAt = time.Now().UTC()
		cp := *a
		return &cp, nil
	}
	a, err := domainasset.New(id, projectID, typ, target, target)
	if err != nil {
		return nil, err
	}
	inv.assets[id] = a
	inv.byValue[key] = id
	cp := *a
	return &cp, nil
}

// Get returns an asset by id.
func (inv *Inventory) Get(id contracts.AssetID) (*domainasset.Asset, error) {
	if inv == nil {
		return nil, contracts.ErrInvalidArgument
	}
	if inv.assetsR != nil {
		return inv.assetsR.GetByID(context.Background(), id)
	}
	inv.mu.RLock()
	a, ok := inv.assets[id]
	if ok {
		cp := *a
		inv.mu.RUnlock()
		return &cp, nil
	}
	inv.mu.RUnlock()
	return nil, contracts.ErrNotFound
}

// GetByValue resolves a canonical project asset by its exact value. It checks
// the in-memory index first, then pages the durable repository so authorization
// does not depend on which assets happened to be loaded into this process.
func (inv *Inventory) GetByValue(projectID contracts.ProjectID, value string) (*domainasset.Asset, error) {
	value = strings.TrimSpace(value)
	if projectID == "" || value == "" {
		return nil, fmt.Errorf("%w: project and asset value required", contracts.ErrInvalidArgument)
	}
	key := string(projectID) + "|" + strings.ToLower(value)
	inv.mu.RLock()
	repo := inv.assetsR
	if repo == nil {
		if id, ok := inv.byValue[key]; ok {
			if a, exists := inv.assets[id]; exists {
				cp := *a
				inv.mu.RUnlock()
				return &cp, nil
			}
		}
	}
	inv.mu.RUnlock()
	if repo == nil {
		return nil, contracts.ErrNotFound
	}
	const pageSize = 250
	var found *domainasset.Asset
	for offset := 0; ; offset += pageSize {
		rows, err := repo.ListByProject(context.Background(), projectID, pageSize, offset)
		if err != nil {
			return nil, err
		}
		for _, a := range rows {
			if a == nil || !strings.EqualFold(strings.TrimSpace(a.Value), value) {
				continue
			}
			if found != nil && found.ID != a.ID {
				return nil, fmt.Errorf("%w: asset value is ambiguous in project", contracts.ErrConflict)
			}
			cp := *a
			found = &cp
		}
		if len(rows) < pageSize {
			break
		}
	}
	if found == nil {
		return nil, contracts.ErrNotFound
	}
	inv.mu.Lock()
	inv.assets[found.ID] = found
	inv.byValue[key] = found.ID
	inv.mu.Unlock()
	return found, nil
}

// ListByProjectPage returns an authoritative bounded project page. Durable
// deployments query PostgreSQL rather than relying on a cold process cache.
func (inv *Inventory) ListByProjectPage(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*domainasset.Asset, error) {
	if inv == nil || projectID == "" || limit < 1 || limit > 500 || offset < 0 {
		return nil, contracts.ErrInvalidArgument
	}
	if inv.assetsR != nil {
		return inv.assetsR.ListByProject(ctx, projectID, limit, offset)
	}
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	var out []*domainasset.Asset
	for _, a := range inv.assets {
		if a.ProjectID == projectID {
			cp := *a
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	if offset >= len(out) {
		return []*domainasset.Asset{}, nil
	}
	out = out[offset:]
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// ListByProject preserves the legacy development/test API.
func (inv *Inventory) ListByProject(projectID contracts.ProjectID) []*domainasset.Asset {
	out, _ := inv.ListByProjectPage(context.Background(), projectID, 500, 0)
	return out
}

// RecordObservation stores a tool observation against an asset.
func (inv *Inventory) RecordObservation(o *domainasset.Observation) error {
	if o == nil {
		return contracts.ErrInvalidArgument
	}
	if err := contracts.ValidateNonEmpty("observation.id", o.ID); err != nil {
		return err
	}
	inv.mu.Lock()
	cp := *o
	inv.obs[o.ID] = &cp
	inv.mu.Unlock()
	if inv.obsR != nil {
		if err := inv.obsR.Upsert(context.Background(), &cp); err != nil {
			inv.mu.Lock()
			delete(inv.obs, o.ID)
			inv.mu.Unlock()
			return err
		}
	}
	return nil
}

// GetObservation returns an observation by id (memory then durable store).
func (inv *Inventory) GetObservation(id string) (*domainasset.Observation, error) {
	inv.mu.RLock()
	o, ok := inv.obs[id]
	if ok {
		cp := *o
		inv.mu.RUnlock()
		return &cp, nil
	}
	inv.mu.RUnlock()
	if inv.obsR != nil {
		return inv.obsR.GetByID(context.Background(), id)
	}
	return nil, contracts.ErrNotFound
}

// ListObservationsByAsset returns observations for an asset.
func (inv *Inventory) ListObservationsByAsset(assetID contracts.AssetID) []*domainasset.Observation {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	var out []*domainasset.Observation
	for _, o := range inv.obs {
		if o.AssetID == assetID {
			cp := *o
			out = append(out, &cp)
		}
	}
	return out
}

// UpsertEndpoint records a service endpoint on an asset.
func (inv *Inventory) UpsertEndpoint(ep *domainasset.ServiceEndpoint) (*domainasset.ServiceEndpoint, error) {
	if ep == nil {
		return nil, contracts.ErrInvalidArgument
	}
	key := fmt.Sprintf("%s|%s|%d|%s", ep.AssetID, ep.Protocol, ep.Port, ep.Path)
	id := "ep-" + shortHash(key)
	inv.mu.Lock()
	if existing, ok := inv.endpoints[id]; ok {
		existing.LastSeen = time.Now().UTC()
		if ep.Banner != "" {
			existing.Banner = ep.Banner
		}
		cp := *existing
		inv.mu.Unlock()
		if inv.endpointsR != nil {
			if err := inv.endpointsR.Upsert(context.Background(), &cp); err != nil {
				return nil, err
			}
		}
		return &cp, nil
	}
	ep.ID = id
	ep.FirstSeen = time.Now().UTC()
	ep.LastSeen = ep.FirstSeen
	cp := *ep
	inv.endpoints[id] = &cp
	inv.mu.Unlock()
	if inv.endpointsR != nil {
		if err := inv.endpointsR.Upsert(context.Background(), &cp); err != nil {
			inv.mu.Lock()
			delete(inv.endpoints, id)
			inv.mu.Unlock()
			return nil, err
		}
	}
	return &cp, nil
}

// GetEndpoint returns an endpoint by id.
func (inv *Inventory) GetEndpoint(id string) (*domainasset.ServiceEndpoint, error) {
	inv.mu.RLock()
	ep, ok := inv.endpoints[id]
	if ok {
		cp := *ep
		inv.mu.RUnlock()
		return &cp, nil
	}
	inv.mu.RUnlock()
	if inv.endpointsR != nil {
		return inv.endpointsR.GetByID(context.Background(), id)
	}
	return nil, contracts.ErrNotFound
}

// ListEndpointsByAsset returns endpoints for an asset.
func (inv *Inventory) ListEndpointsByAsset(assetID contracts.AssetID) []*domainasset.ServiceEndpoint {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	var out []*domainasset.ServiceEndpoint
	for _, e := range inv.endpoints {
		if e.AssetID == assetID {
			cp := *e
			out = append(out, &cp)
		}
	}
	return out
}

// IngestFromResult upserts target asset + observations from NormalizedResult.
// Does not authorize execution and does not create findings (caller may).
func (inv *Inventory) IngestFromResult(
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
	taskID contracts.TaskID,
	executionID string,
	result tools.NormalizedResult,
) (*domainasset.Asset, []*domainasset.Observation, error) {
	target, _ := result.Raw["target"].(string)
	if target == "" {
		target, _ = result.Raw["url"].(string)
	}
	if target == "" {
		target = "unknown"
	}
	a, err := inv.UpsertTarget(projectID, target, inferType(target))
	if err != nil {
		return nil, nil, err
	}
	var obs []*domainasset.Observation
	now := time.Now().UTC()
	summary := result.Summary
	if summary == "" {
		summary = "tool observation"
	}
	oid := "obs-" + shortHash(string(a.ID)+"|"+result.ToolID+"|"+summary+"|"+now.Format(time.RFC3339Nano))
	o, err := domainasset.NewObservation(oid, a.ID, projectID, contracts.ObservationKindGeneric, summary)
	if err != nil {
		return a, nil, err
	}
	o.ScopeID = scopeID
	o.TaskID = taskID
	o.ExecutionID = executionID
	o.ToolID = result.ToolID
	o.CapabilityID = contracts.CapabilityID(result.Capability)
	o.Detail = summary
	if err := inv.RecordObservation(o); err != nil {
		return a, nil, fmt.Errorf("persist asset observation: %w", err)
	}
	obs = append(obs, o)

	// Optional endpoint from findings detail like "22/tcp"
	for _, h := range result.Findings {
		if strings.Contains(h.Detail, "/tcp") || strings.Contains(h.Detail, "/udp") {
			port, proto := parsePortProto(h.Detail)
			if port > 0 {
				ep := &domainasset.ServiceEndpoint{
					AssetID: a.ID, ProjectID: projectID, Protocol: proto, Port: port, Banner: h.Title,
				}
				if _, err := inv.UpsertEndpoint(ep); err != nil {
					return a, obs, fmt.Errorf("persist service endpoint: %w", err)
				}
			}
		}
	}
	return a, obs, nil
}

func inferType(target string) contracts.AssetType {
	t := strings.ToLower(target)
	if strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://") {
		return contracts.AssetTypeURL
	}
	if strings.Contains(t, ".") && !isIPv4(t) {
		return contracts.AssetTypeDomain
	}
	return contracts.AssetTypeHost
}

func isIPv4(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

func parsePortProto(detail string) (int, string) {
	detail = strings.TrimSpace(detail)
	proto := "tcp"
	if strings.Contains(detail, "/udp") {
		proto = "udp"
	}
	var port int
	fmt.Sscanf(detail, "%d/", &port)
	return port, proto
}

func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:8])
}

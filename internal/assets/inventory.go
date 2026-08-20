// Package assets provides the Asset & Target Intelligence inventory.
// Observations feed findings; inventory never authorizes execution.
package assets

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"cyberos/internal/contracts"
	domainasset "cyberos/internal/domain/asset"
	"cyberos/internal/tools"
)

// Inventory is the in-process asset/target intelligence store.
type Inventory struct {
	mu        sync.RWMutex
	assets    map[contracts.AssetID]*domainasset.Asset
	byValue   map[string]contracts.AssetID // project|value → asset id
	endpoints map[string]*domainasset.ServiceEndpoint
	obs       map[string]*domainasset.Observation
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
	inv.mu.Lock()
	defer inv.mu.Unlock()
	if id, ok := inv.byValue[key]; ok {
		a := inv.assets[id]
		a.UpdatedAt = time.Now().UTC()
		cp := *a
		return &cp, nil
	}
	id := contracts.AssetID("asset-" + shortHash(key))
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
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	a, ok := inv.assets[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *a
	return &cp, nil
}

// ListByProject returns assets for a project.
func (inv *Inventory) ListByProject(projectID contracts.ProjectID) []*domainasset.Asset {
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	var out []*domainasset.Asset
	for _, a := range inv.assets {
		if a.ProjectID == projectID {
			cp := *a
			out = append(out, &cp)
		}
	}
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
	defer inv.mu.Unlock()
	cp := *o
	inv.obs[o.ID] = &cp
	return nil
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
	defer inv.mu.Unlock()
	if existing, ok := inv.endpoints[id]; ok {
		existing.LastSeen = time.Now().UTC()
		if ep.Banner != "" {
			existing.Banner = ep.Banner
		}
		cp := *existing
		return &cp, nil
	}
	ep.ID = id
	ep.FirstSeen = time.Now().UTC()
	ep.LastSeen = ep.FirstSeen
	cp := *ep
	inv.endpoints[id] = &cp
	return &cp, nil
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
	_ = inv.RecordObservation(o)
	obs = append(obs, o)

	// Optional endpoint from findings detail like "22/tcp"
	for _, h := range result.Findings {
		if strings.Contains(h.Detail, "/tcp") || strings.Contains(h.Detail, "/udp") {
			port, proto := parsePortProto(h.Detail)
			if port > 0 {
				ep := &domainasset.ServiceEndpoint{
					AssetID: a.ID, ProjectID: projectID, Protocol: proto, Port: port, Banner: h.Title,
				}
				_, _ = inv.UpsertEndpoint(ep)
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

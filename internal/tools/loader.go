package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"cyberos/internal/contracts"
)

// Catalog holds discovery-only tool descriptors loaded from packs.
// It never executes tools or modifies process allowlists.
type Catalog struct {
	mu    sync.RWMutex
	tools map[string]Descriptor
}

// NewCatalog constructs an empty discovery catalog.
func NewCatalog() *Catalog {
	return &Catalog{tools: make(map[string]Descriptor)}
}

// Get returns a tool descriptor by id.
func (c *Catalog) Get(id string) (Descriptor, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	d, ok := c.tools[id]
	if !ok {
		return Descriptor{}, contracts.ErrNotFound
	}
	return d, nil
}

// List returns all discovered tool descriptors.
func (c *Catalog) List() []Descriptor {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]Descriptor, 0, len(c.tools))
	for _, d := range c.tools {
		out = append(out, d)
	}
	return out
}

func (c *Catalog) putAll(ds []Descriptor) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tools == nil {
		c.tools = make(map[string]Descriptor)
	}
	for _, d := range ds {
		c.tools[d.ID] = d
	}
}

func (c *Catalog) removeIDs(ids []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range ids {
		delete(c.tools, id)
	}
}

// Loader installs pack files into a Platform (discovery only).
type Loader struct {
	Platform *Platform
	Catalog  *Catalog
	// SignerKeys and VerifyMode are optional Phase 18B verification (not authorization).
	SignerKeys *SignerKeyRegistry
	VerifyMode VerificationMode
}

// NewLoader constructs a pack loader. If cat is nil, the platform catalog is used.
func NewLoader(p *Platform, cat *Catalog) *Loader {
	if cat == nil && p != nil {
		cat = p.Catalog()
	}
	if cat == nil {
		cat = NewCatalog()
	}
	return &Loader{Platform: p, Catalog: cat}
}

// InstallResult is the outcome of a pack install.
type InstallResult struct {
	PackID  string
	Version contracts.SchemaVersion
	State   PackState
	ToolIDs []string
}

// InstallPackFile validates and registers a pack file atomically (all or nothing).
// Does not execute tools, change process policy, or issue grants.
func (l *Loader) catalog() *Catalog {
	if l.Catalog != nil {
		return l.Catalog
	}
	if l.Platform != nil && l.Platform.Catalog() != nil {
		return l.Platform.Catalog()
	}
	l.Catalog = NewCatalog()
	return l.Catalog
}

func (l *Loader) InstallPackFile(pf PackFile) (*InstallResult, error) {
	if l == nil || l.Platform == nil {
		return nil, fmt.Errorf("%w: loader not configured", contracts.ErrInvalidArgument)
	}
	cat := l.catalog()
	if err := pf.Validate(); err != nil {
		return nil, err
	}
	// Phase 18A/18B: integrity/signature verification (not authorization).
	mode := l.VerifyMode
	if mode == "" {
		mode = VerifyLegacy
	}
	if mode == VerifyLegacy {
		if err := VerifyDigest(pf, pf.Pack.ContentDigest); err != nil {
			// VerifyDigest compares including legacy path; recompute without meta fields
			tmp := pf
			tmp.Pack.ContentDigest = ""
			tmp.Signature = nil
			if pf.Pack.ContentDigest != "" {
				if err2 := VerifyDigest(tmp, pf.Pack.ContentDigest); err2 != nil {
					return nil, err2
				}
			}
		}
	} else {
		res := VerifyPack(pf, l.SignerKeys, mode)
		if !res.Valid {
			return nil, fmt.Errorf("%w: pack verification failed: %s", contracts.ErrInvalidArgument, res.Reason)
		}
	}
	// Dependency check against already-installed packs/tools (no download).
	if pf.Pack.Ops != nil {
		if err := CheckDependencies(l.Platform, pf.Pack.Ops.Dependencies); err != nil {
			return nil, err
		}
	}
	for _, t := range pf.Tools {
		if t.Ops != nil {
			if err := CheckDependencies(l.Platform, t.Ops.Dependencies); err != nil {
				return nil, err
			}
		}
	}
	pack := pf.Pack
	if pack.State == "" {
		pack.State = PackStateRegistered
	}
	if len(pack.ToolIDs) == 0 && len(pf.Tools) > 0 {
		for _, t := range pf.Tools {
			pack.ToolIDs = append(pack.ToolIDs, t.ID)
		}
	}
	var caps []contracts.CapabilityID
	for _, t := range pf.Tools {
		caps = append(caps, t.Capabilities...)
	}
	if len(pack.CapabilityIDs) == 0 {
		pack.CapabilityIDs = caps
	}

	// Atomic: register pack first; on tool catalog conflict with different version, reject.
	var descs []Descriptor
	for _, t := range pf.Tools {
		d := t.Descriptor()
		if existing, err := cat.Get(d.ID); err == nil {
			if existing.Version != d.Version {
				return nil, fmt.Errorf("%w: tool %s already registered at version %s", contracts.ErrConflict, d.ID, existing.Version)
			}
		}
		descs = append(descs, d)
	}

	if err := l.Platform.RegisterPack(pack); err != nil {
		return nil, err
	}
	cat.putAll(descs)

	return &InstallResult{
		PackID: pack.ID, Version: pack.Version, State: pack.State, ToolIDs: append([]string{}, pack.ToolIDs...),
	}, nil
}

// LoadJSONFile reads a pack JSON file from a controlled path and installs it.
func (l *Loader) LoadJSONFile(path string) (*InstallResult, error) {
	path = filepath.Clean(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: read pack file: %v", contracts.ErrInvalidArgument, err)
	}
	var pf PackFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("%w: parse pack json: %v", contracts.ErrInvalidArgument, err)
	}
	return l.InstallPackFile(pf)
}

// LoadDirectory loads all *.pack.json files from a directory (deterministic sorted order).
func (l *Loader) LoadDirectory(dir string) ([]InstallResult, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) == ".json" {
			names = append(names, name)
		}
	}
	// sorted for deterministic load
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	var out []InstallResult
	for _, name := range names {
		res, err := l.LoadJSONFile(filepath.Join(dir, name))
		if err != nil {
			return out, fmt.Errorf("%s: %w", name, err)
		}
		out = append(out, *res)
	}
	return out, nil
}

// SetPackState updates pack lifecycle state (metadata only).
func (l *Loader) SetPackState(packID string, state PackState) error {
	if l == nil || l.Platform == nil {
		return fmt.Errorf("%w: loader not configured", contracts.ErrInvalidArgument)
	}
	return l.Platform.Packs().SetState(packID, state)
}

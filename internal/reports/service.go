// Package reports provides assessment report generation (management only).
package reports

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"cyberos/internal/cases"
	"cyberos/internal/contracts"
	"cyberos/internal/riskprio"
	"cyberos/internal/secrets"
	"cyberos/internal/storage/ports"
)

const SchemaVersion = "cyberos-report-v1"

// Report is metadata + snapshot content for export.
type Report struct {
	ID            string
	TenantID      contracts.TenantID
	ProjectID     contracts.ProjectID
	AssessmentID  string
	GeneratedAt   time.Time
	GeneratedBy   contracts.PrincipalID
	SchemaVersion string
	Status        string
	// Snapshot is the versioned content used by exporters.
	Snapshot *Snapshot
}

// Snapshot is the stable report body.
type Snapshot struct {
	SchemaVersion string           `json:"schema_version"`
	AssessmentID  string           `json:"assessment_id"`
	ProjectID     string           `json:"project_id"`
	GeneratedAt   time.Time        `json:"generated_at"`
	Executive     string           `json:"executive_summary"`
	Assets        []map[string]any `json:"assets"`
	Findings      []map[string]any `json:"findings"`
	Risks         []map[string]any `json:"risks"`
	Cases         []map[string]any `json:"cases"`
	RiskSummary   map[string]any   `json:"risk_summary"`
	EvidenceRefs  []string         `json:"evidence_refs"`
	Timeline      []string         `json:"timeline"`
	Extra         map[string]any   `json:"extra,omitempty"`
}

// BuildInput is data assembled for a report (caller provides already-loaded data).
type BuildInput struct {
	AssessmentID string
	ProjectID    contracts.ProjectID
	TenantID     contracts.TenantID
	GeneratedBy  contracts.PrincipalID
	Assets       []map[string]any
	Findings     []map[string]any
	Risks        []map[string]any
	Cases        []*cases.Case
	RiskSummary  riskprio.Summary
	EvidenceRefs []string
	Timeline     []string
}

// Service stores report metadata.
type Service struct {
	mu    sync.RWMutex
	items map[string]*Report
	repo  ports.ReportRepository
}

// NewService constructs a report service.
func NewService() *Service {
	return &Service{items: map[string]*Report{}}
}

// WithRepository attaches optional durable persistence (Phase 7B).
func (s *Service) WithRepository(repo ports.ReportRepository) *Service {
	if s != nil {
		s.repo = repo
	}
	return s
}

// Create builds and stores a report from BuildInput.
func (s *Service) Create(id string, in BuildInput) (*Report, error) {
	if id == "" || in.AssessmentID == "" {
		return nil, fmt.Errorf("%w: id and assessment required", contracts.ErrInvalidArgument)
	}
	now := time.Now().UTC()
	exec := fmt.Sprintf("Assessment %s: %d findings, %d assets, highest risk score %d.",
		in.AssessmentID, len(in.Findings), len(in.Assets), in.RiskSummary.HighestScore)
	caseMaps := make([]map[string]any, 0, len(in.Cases))
	for _, c := range in.Cases {
		if c == nil {
			continue
		}
		caseMaps = append(caseMaps, map[string]any{
			"id": c.ID, "title": c.Title, "status": c.Status, "priority": c.Priority,
			"finding_ids": c.FindingIDs, "asset_ids": c.AssetIDs,
		})
	}
	snap := &Snapshot{
		SchemaVersion: SchemaVersion,
		AssessmentID:  in.AssessmentID,
		ProjectID:     string(in.ProjectID),
		GeneratedAt:   now,
		Executive:     exec,
		Assets:        redactMaps(in.Assets),
		Findings:      redactMaps(in.Findings),
		Risks:         redactMaps(in.Risks),
		Cases:         caseMaps,
		RiskSummary: map[string]any{
			"total": in.RiskSummary.Total, "by_priority": in.RiskSummary.ByPriority,
			"highest_score": in.RiskSummary.HighestScore, "model": in.RiskSummary.ModelVersion,
		},
		EvidenceRefs: append([]string{}, in.EvidenceRefs...),
		Timeline:     append([]string{}, in.Timeline...),
	}
	r := &Report{
		ID: id, TenantID: in.TenantID, ProjectID: in.ProjectID, AssessmentID: in.AssessmentID,
		GeneratedAt: now, GeneratedBy: in.GeneratedBy, SchemaVersion: SchemaVersion,
		Status: "READY", Snapshot: snap,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; ok {
		return nil, contracts.ErrConflict
	}
	if s.repo != nil {
		snap, err := json.Marshal(r.Snapshot)
		if err != nil {
			return nil, err
		}
		if err := s.repo.Upsert(context.Background(), ports.ReportRecord{
			ID: r.ID, TenantID: r.TenantID, ProjectID: r.ProjectID, AssessmentID: r.AssessmentID,
			GeneratedAt: r.GeneratedAt, GeneratedBy: r.GeneratedBy, SchemaVersion: r.SchemaVersion,
			Status: r.Status, SnapshotJSON: snap,
		}); err != nil {
			return nil, err
		}
	}
	s.items[id] = r
	cp := *r
	return &cp, nil
}

// Get returns a report by id.
func (s *Service) Get(id string) (*Report, error) {
	s.mu.RLock()
	r, ok := s.items[id]
	if ok {
		cp := *r
		s.mu.RUnlock()
		return &cp, nil
	}
	s.mu.RUnlock()
	if s.repo != nil {
		rec, err := s.repo.GetByID(context.Background(), id)
		if err != nil {
			return nil, err
		}
		var snap Snapshot
		if len(rec.SnapshotJSON) > 0 {
			_ = json.Unmarshal(rec.SnapshotJSON, &snap)
		}
		return &Report{
			ID: rec.ID, TenantID: rec.TenantID, ProjectID: rec.ProjectID, AssessmentID: rec.AssessmentID,
			GeneratedAt: rec.GeneratedAt, GeneratedBy: rec.GeneratedBy, SchemaVersion: rec.SchemaVersion,
			Status: rec.Status, Snapshot: &snap,
		}, nil
	}
	return nil, contracts.ErrNotFound
}

// List returns all reports (test/demo).
func (s *Service) List() []*Report {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Report, 0, len(s.items))
	for _, r := range s.items {
		cp := *r
		out = append(out, &cp)
	}
	return out
}

// Export renders a report in the requested format.
func (s *Service) Export(id string, format contracts.ReportFormat) ([]byte, error) {
	r, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	switch format {
	case contracts.ReportFormatJSON:
		return exportJSON(r.Snapshot)
	case contracts.ReportFormatMarkdown:
		return exportMarkdown(r.Snapshot), nil
	default:
		return nil, fmt.Errorf("%w: unsupported format", contracts.ErrInvalidArgument)
	}
}

func exportJSON(snap *Snapshot) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snap); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func exportMarkdown(snap *Snapshot) []byte {
	var b strings.Builder
	b.WriteString("# CYBEROS Security Assessment Report\n\n")
	b.WriteString("Schema: " + snap.SchemaVersion + "\n\n")
	b.WriteString("## Executive Summary\n\n")
	b.WriteString(snap.Executive + "\n\n")
	b.WriteString("## Assessment\n\n")
	b.WriteString(fmt.Sprintf("- Assessment ID: %s\n- Project ID: %s\n- Generated: %s\n\n",
		snap.AssessmentID, snap.ProjectID, snap.GeneratedAt.UTC().Format(time.RFC3339)))
	b.WriteString("## Assets\n\n")
	b.WriteString(fmt.Sprintf("%d asset(s)\n\n", len(snap.Assets)))
	b.WriteString("## Findings\n\n")
	b.WriteString(fmt.Sprintf("%d finding(s)\n\n", len(snap.Findings)))
	b.WriteString("## Risk Summary\n\n")
	keys := make([]string, 0, len(snap.RiskSummary))
	for k := range snap.RiskSummary {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("- %s: %v\n", k, snap.RiskSummary[k]))
	}
	b.WriteString("\n## Security Cases\n\n")
	b.WriteString(fmt.Sprintf("%d case(s)\n\n", len(snap.Cases)))
	b.WriteString("## Evidence / Provenance\n\n")
	for _, e := range snap.EvidenceRefs {
		b.WriteString("- " + e + "\n")
	}
	b.WriteString("\n## Timeline\n\n")
	for _, t := range snap.Timeline {
		b.WriteString("- " + t + "\n")
	}
	return []byte(b.String())
}

func redactMaps(in []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(in))
	for _, m := range in {
		cp := map[string]any{}
		for k, v := range m {
			if str, ok := v.(string); ok {
				cp[k] = secrets.RedactString(str)
			} else {
				cp[k] = v
			}
		}
		out = append(out, cp)
	}
	return out
}

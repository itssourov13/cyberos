// Package riskprio is the CYBEROS Risk & Prioritization foundation.
// It is downstream analysis only — never authorization or execution.
package riskprio

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"cyberos/internal/contracts"
	domainfinding "cyberos/internal/domain/finding"
	"cyberos/internal/storage/ports"
)

const ModelVersion = "cyberos-risk-v1"

// Evaluation is a durable, explainable prioritization record.
type Evaluation struct {
	ID               string
	ProjectID        contracts.ProjectID
	AssessmentID     string
	FindingID        contracts.FindingID
	AssetID          contracts.AssetID
	Severity         contracts.FindingSeverity
	Confidence       contracts.FindingConfidence
	AssetCriticality contracts.AssetCriticality
	Exposure         contracts.ExposureContext
	Score            int
	Priority         contracts.RiskPriority
	Factors          map[string]string
	ModelVersion     string
	CalculatedAt     time.Time
}

// Input is the deterministic scoring input (no timestamps).
type Input struct {
	FindingID        contracts.FindingID
	ProjectID        contracts.ProjectID
	AssessmentID     string
	AssetID          contracts.AssetID
	Severity         contracts.FindingSeverity
	Confidence       contracts.FindingConfidence
	AssetCriticality contracts.AssetCriticality
	Exposure         contracts.ExposureContext
}

// Engine calculates and stores risk evaluations.
type Engine struct {
	mu    sync.RWMutex
	items map[string]*Evaluation // by finding id (latest)
	all   map[string]*Evaluation // by risk id
	repo  ports.RiskEvaluationRepository
}

// NewEngine constructs a prioritization engine.
func NewEngine() *Engine {
	return &Engine{
		items: map[string]*Evaluation{},
		all:   map[string]*Evaluation{},
	}
}

// WithRepository attaches optional durable persistence (Phase 7A).
func (e *Engine) WithRepository(repo ports.RiskEvaluationRepository) *Engine {
	if e != nil {
		e.repo = repo
	}
	return e
}

func (e *Engine) persist(ev *Evaluation) error {
	if e == nil || e.repo == nil || ev == nil {
		return nil
	}
	return e.repo.Upsert(context.Background(), ports.RiskEvaluationRecord{
		ID: ev.ID, ProjectID: ev.ProjectID, AssessmentID: ev.AssessmentID,
		FindingID: ev.FindingID, AssetID: ev.AssetID, Severity: ev.Severity,
		Confidence: ev.Confidence, AssetCriticality: ev.AssetCriticality, Exposure: ev.Exposure,
		Score: ev.Score, Priority: ev.Priority, Factors: ev.Factors,
		ModelVersion: ev.ModelVersion, CalculatedAt: ev.CalculatedAt,
	})
}

// Score computes a deterministic 0–100 score and priority from factors.
// Weights are documented: severity 40%, criticality 25%, exposure 20%, confidence 15%.
func Score(in Input) (int, contracts.RiskPriority, map[string]string) {
	sev := severityPoints(in.Severity)
	crit := criticalityPoints(in.AssetCriticality)
	exp := exposurePoints(in.Exposure)
	conf := confidencePoints(in.Confidence)
	// Weighted sum (max 100)
	score := (sev*40 + crit*25 + exp*20 + conf*15) / 100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	factors := map[string]string{
		"severity":           string(in.Severity),
		"severity_points":    fmt.Sprintf("%d", sev),
		"confidence":         string(in.Confidence),
		"confidence_points":  fmt.Sprintf("%d", conf),
		"asset_criticality":  string(in.AssetCriticality),
		"criticality_points": fmt.Sprintf("%d", crit),
		"exposure":           string(in.Exposure),
		"exposure_points":    fmt.Sprintf("%d", exp),
		"model":              ModelVersion,
		"weights":            "severity:40,criticality:25,exposure:20,confidence:15",
	}
	return score, priorityFromScore(score), factors
}

// Evaluate calculates and stores risk for a finding.
func (e *Engine) Evaluate(in Input) (*Evaluation, error) {
	if e == nil {
		return nil, contracts.ErrInvalidArgument
	}
	if in.FindingID == "" {
		return nil, fmt.Errorf("%w: finding id required", contracts.ErrInvalidArgument)
	}
	if !in.Severity.Valid() {
		in.Severity = contracts.FindingSeverityInfo
	}
	if !in.Confidence.Valid() {
		in.Confidence = contracts.FindingConfidenceMedium
	}
	if !in.AssetCriticality.Valid() {
		in.AssetCriticality = contracts.AssetCriticalityUnknown
	}
	if !in.Exposure.Valid() {
		in.Exposure = contracts.ExposureUnknown
	}
	score, prio, factors := Score(in)
	ev := &Evaluation{
		ID:               "risk-" + string(in.FindingID),
		ProjectID:        in.ProjectID,
		AssessmentID:     in.AssessmentID,
		FindingID:        in.FindingID,
		AssetID:          in.AssetID,
		Severity:         in.Severity,
		Confidence:       in.Confidence,
		AssetCriticality: in.AssetCriticality,
		Exposure:         in.Exposure,
		Score:            score,
		Priority:         prio,
		Factors:          factors,
		ModelVersion:     ModelVersion,
		CalculatedAt:     time.Now().UTC(),
	}
	if err := e.persist(ev); err != nil {
		return nil, fmt.Errorf("persist risk evaluation %s: %w", ev.ID, err)
	}
	if e.repo == nil {
		e.mu.Lock()
		e.items[string(in.FindingID)] = ev
		e.all[ev.ID] = ev
		e.mu.Unlock()
	}
	cp := *ev
	return &cp, nil
}

// EvaluateFinding maps a domain finding (+ optional criticality/exposure) to risk.
func (e *Engine) EvaluateFinding(f *domainfinding.Finding, crit contracts.AssetCriticality, exposure contracts.ExposureContext, assetID contracts.AssetID) (*Evaluation, error) {
	if f == nil {
		return nil, contracts.ErrInvalidArgument
	}
	conf := f.Confidence
	if !conf.Valid() {
		conf = contracts.FindingConfidenceMedium
	}
	return e.Evaluate(Input{
		FindingID: f.ID, ProjectID: f.ProjectID, AssessmentID: f.AssessmentID,
		AssetID: assetID, Severity: f.Severity, Confidence: conf,
		AssetCriticality: crit, Exposure: exposure,
	})
}

// Get returns evaluation by risk id.
func (e *Engine) Get(id string) (*Evaluation, error) {
	if e.repo != nil {
		rec, err := e.repo.GetByID(context.Background(), id)
		if err != nil {
			return nil, err
		}
		return recordToEval(rec), nil
	}
	e.mu.RLock()
	ev, ok := e.all[id]
	if ok {
		cp := *ev
		e.mu.RUnlock()
		return &cp, nil
	}
	e.mu.RUnlock()
	return nil, contracts.ErrNotFound
}

// GetByFinding returns the latest evaluation for a finding.
func (e *Engine) GetByFinding(id contracts.FindingID) (*Evaluation, error) {
	if e.repo != nil {
		rec, err := e.repo.GetByFinding(context.Background(), id)
		if err != nil {
			return nil, err
		}
		return recordToEval(rec), nil
	}
	e.mu.RLock()
	ev, ok := e.items[string(id)]
	if ok {
		cp := *ev
		e.mu.RUnlock()
		return &cp, nil
	}
	e.mu.RUnlock()
	return nil, contracts.ErrNotFound
}

func recordToEval(rec ports.RiskEvaluationRecord) *Evaluation {
	return &Evaluation{
		ID: rec.ID, ProjectID: rec.ProjectID, AssessmentID: rec.AssessmentID,
		FindingID: rec.FindingID, AssetID: rec.AssetID, Severity: rec.Severity,
		Confidence: rec.Confidence, AssetCriticality: rec.AssetCriticality, Exposure: rec.Exposure,
		Score: rec.Score, Priority: rec.Priority, Factors: rec.Factors,
		ModelVersion: rec.ModelVersion, CalculatedAt: rec.CalculatedAt,
	}
}

// ListByProjectPage reads an authoritative project page from PostgreSQL when
// configured; durable read errors are returned to the API instead of hidden.
func (e *Engine) ListByProjectPage(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*Evaluation, error) {
	if e == nil || projectID == "" || limit < 1 || limit > 500 || offset < 0 {
		return nil, contracts.ErrInvalidArgument
	}
	if e.repo != nil {
		rows, err := e.repo.ListByProject(ctx, projectID, limit, offset)
		if err != nil {
			return nil, err
		}
		out := make([]*Evaluation, 0, len(rows))
		for _, row := range rows {
			out = append(out, recordToEval(row))
		}
		return out, nil
	}
	return pageEvaluations(e.filter(func(ev *Evaluation) bool { return ev.ProjectID == projectID }), limit, offset), nil
}

// ListByAssessmentPage is the durable assessment-scoped read path.
func (e *Engine) ListByAssessmentPage(ctx context.Context, assessmentID string, limit, offset int) ([]*Evaluation, error) {
	if e == nil || assessmentID == "" || limit < 1 || limit > 500 || offset < 0 {
		return nil, contracts.ErrInvalidArgument
	}
	if e.repo != nil {
		rows, err := e.repo.ListByAssessment(ctx, assessmentID, limit, offset)
		if err != nil {
			return nil, err
		}
		out := make([]*Evaluation, 0, len(rows))
		for _, row := range rows {
			out = append(out, recordToEval(row))
		}
		return out, nil
	}
	return pageEvaluations(e.filter(func(ev *Evaluation) bool { return ev.AssessmentID == assessmentID }), limit, offset), nil
}

// ListByProject returns the compatibility page.
func (e *Engine) ListByProject(projectID contracts.ProjectID) []*Evaluation {
	out, _ := e.ListByProjectPage(context.Background(), projectID, 500, 0)
	return out
}

// ListByAssessment returns the compatibility page.
func (e *Engine) ListByAssessment(assessmentID string) []*Evaluation {
	out, _ := e.ListByAssessmentPage(context.Background(), assessmentID, 500, 0)
	return out
}

func pageEvaluations(rows []*Evaluation, limit, offset int) []*Evaluation {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].CalculatedAt.Equal(rows[j].CalculatedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].CalculatedAt.After(rows[j].CalculatedAt)
	})
	if offset >= len(rows) {
		return []*Evaluation{}
	}
	rows = rows[offset:]
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}

// ListByAsset returns evaluations for an asset.
func (e *Engine) ListByAsset(assetID contracts.AssetID) []*Evaluation {
	return e.filter(func(ev *Evaluation) bool { return ev.AssetID == assetID })
}

// Summary is a deterministic assessment risk rollup.
type Summary struct {
	AssessmentID string         `json:"assessment_id"`
	Total        int            `json:"total"`
	ByPriority   map[string]int `json:"by_priority"`
	HighestScore int            `json:"highest_score"`
	ModelVersion string         `json:"model_version"`
}

// SummarizeAssessmentContext reads every durable page so a restart does not
// reset the risk summary to zero or silently cap it at the first 500 rows.
func (e *Engine) SummarizeAssessmentContext(ctx context.Context, assessmentID string) (Summary, error) {
	if e == nil || assessmentID == "" {
		return Summary{}, contracts.ErrInvalidArgument
	}
	s := Summary{
		AssessmentID: assessmentID,
		ByPriority: map[string]int{
			"INFO": 0, "LOW": 0, "MEDIUM": 0, "HIGH": 0, "CRITICAL": 0,
		},
		ModelVersion: ModelVersion,
	}
	for offset := 0; ; offset += 500 {
		rows, err := e.ListByAssessmentPage(ctx, assessmentID, 500, offset)
		if err != nil {
			return Summary{}, err
		}
		for _, ev := range rows {
			s.Total++
			s.ByPriority[string(ev.Priority)]++
			if ev.Score > s.HighestScore {
				s.HighestScore = ev.Score
			}
		}
		if len(rows) < 500 {
			break
		}
	}
	return s, nil
}

// SummarizeAssessment is the compatibility helper for local tools.
func (e *Engine) SummarizeAssessment(assessmentID string) Summary {
	summary, _ := e.SummarizeAssessmentContext(context.Background(), assessmentID)
	return summary
}

func (e *Engine) filter(pred func(*Evaluation) bool) []*Evaluation {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var out []*Evaluation
	for _, ev := range e.all {
		if pred(ev) {
			cp := *ev
			out = append(out, &cp)
		}
	}
	return out
}

func severityPoints(s contracts.FindingSeverity) int {
	switch s {
	case contracts.FindingSeverityInfo:
		return 10
	case contracts.FindingSeverityLow:
		return 30
	case contracts.FindingSeverityMedium:
		return 55
	case contracts.FindingSeverityHigh:
		return 80
	case contracts.FindingSeverityCritical:
		return 100
	default:
		return 10
	}
}

func criticalityPoints(c contracts.AssetCriticality) int {
	switch c {
	case contracts.AssetCriticalityUnknown, contracts.AssetCriticalityLow:
		return 20
	case contracts.AssetCriticalityMedium:
		return 50
	case contracts.AssetCriticalityHigh:
		return 80
	case contracts.AssetCriticalityCritical:
		return 100
	default:
		return 20
	}
}

func exposurePoints(e contracts.ExposureContext) int {
	switch e {
	case contracts.ExposureUnknown, contracts.ExposureInternal:
		return 20
	case contracts.ExposureExternal:
		return 60
	case contracts.ExposureInternetFacing:
		return 100
	default:
		return 20
	}
}

func confidencePoints(c contracts.FindingConfidence) int {
	switch c {
	case contracts.FindingConfidenceLow:
		return 30
	case contracts.FindingConfidenceMedium:
		return 60
	case contracts.FindingConfidenceHigh:
		return 100
	default:
		return 60
	}
}

func priorityFromScore(score int) contracts.RiskPriority {
	switch {
	case score >= 85:
		return contracts.RiskPriorityCritical
	case score >= 70:
		return contracts.RiskPriorityHigh
	case score >= 45:
		return contracts.RiskPriorityMedium
	case score >= 25:
		return contracts.RiskPriorityLow
	default:
		return contracts.RiskPriorityInfo
	}
}

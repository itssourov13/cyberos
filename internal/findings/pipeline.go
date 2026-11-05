package findings

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"cyberos/internal/contracts"
	domainfinding "cyberos/internal/domain/finding"
	"cyberos/internal/tools"
)

// IngestRequest maps a NormalizedResult into findings (not authorization).
type IngestRequest struct {
	Result          tools.NormalizedResult
	ProjectID       contracts.ProjectID
	ScopeID         contracts.ScopeID
	TaskID          contracts.TaskID
	AssessmentID    string
	ExecutionID     string
	PrincipalID     contracts.PrincipalID
	CorrelationID   contracts.CorrelationID
	EvidenceIDs     []contracts.EvidenceID
	DefaultSeverity contracts.FindingSeverity
}

// IngestResult describes what happened for one finding.
type IngestResult struct {
	Finding *domainfinding.Finding
	Action  contracts.FindingStatus // NEW, UNCHANGED, UPDATED, REOPENED
}

// Ingest creates or updates findings from a NormalizedResult using fingerprints.
func (s *Service) Ingest(req IngestRequest) ([]IngestResult, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: findings service not configured", contracts.ErrInvalidArgument)
	}
	sev := req.DefaultSeverity
	if !sev.Valid() {
		sev = contracts.FindingSeverityInfo
	}
	hints := req.Result.Findings
	if len(hints) == 0 && req.Result.Summary != "" {
		hints = []tools.FindingHint{{Title: req.Result.Summary, Severity: string(sev)}}
	}
	var out []IngestResult
	target, _ := req.Result.Raw["target"].(string)
	if target == "" {
		target, _ = req.Result.Raw["url"].(string)
	}
	for i, h := range hints {
		title := h.Title
		if title == "" {
			title = req.Result.Summary
		}
		if title == "" {
			title = "observation"
		}
		hs := parseSeverity(h.Severity, sev)
		fp := Fingerprint(string(req.ProjectID), target, req.Result.Capability, title, h.Detail)
		var existing *domainfinding.Finding
		if s.repo != nil {
			loaded, err := s.repo.GetByFingerprint(context.Background(), req.ProjectID, fp)
			switch {
			case err == nil:
				existing = loaded
			case errors.Is(err, contracts.ErrNotFound):
				// No existing finding for this fingerprint.
			case err != nil:
				return out, fmt.Errorf("lookup finding fingerprint: %w", err)
			}
		} else {
			s.mu.RLock()
			existing = s.byFingerprint[fp]
			s.mu.RUnlock()
		}
		now := time.Now().UTC()
		if existing != nil {
			// Never mutate a repository-returned or shared cache object in place.
			cp := *existing
			existing = &cp
			action := contracts.FindingStatusUnchanged
			if existing.Status == contracts.FindingStatusResolved {
				existing.Status = contracts.FindingStatusReopened
				action = contracts.FindingStatusReopened
			} else if h.Detail != "" && h.Detail != existing.Description {
				existing.Description = h.Detail
				existing.Status = contracts.FindingStatusUpdated
				action = contracts.FindingStatusUpdated
			}
			existing.LastSeen = now
			if len(req.EvidenceIDs) > 0 {
				existing.EvidenceIDs = appendUnique(existing.EvidenceIDs, req.EvidenceIDs...)
			}
			if err := s.persist(existing); err != nil {
				return out, fmt.Errorf("persist updated finding %s: %w", existing.ID, err)
			}
			if s.repo == nil {
				s.mu.Lock()
				s.items[existing.ID] = existing
				s.byFingerprint[fp] = existing
				s.mu.Unlock()
			}
			out = append(out, IngestResult{Finding: existing, Action: action})
			continue
		}
		id := contracts.FindingID(fmt.Sprintf("f-%s-%d", fp[:8], i))
		f, err := domainfinding.New(id, req.TaskID, req.ProjectID, req.ScopeID, hs, title)
		if err != nil {
			return out, err
		}
		f.Description = h.Detail
		f.Fingerprint = fp
		f.Target = target
		f.Location = h.Detail
		f.ToolID = req.Result.ToolID
		f.CapabilityID = contracts.CapabilityID(req.Result.Capability)
		f.AssessmentID = req.AssessmentID
		f.ExecutionID = req.ExecutionID
		f.PrincipalID = req.PrincipalID
		f.CorrelationID = req.CorrelationID
		f.EvidenceIDs = append([]contracts.EvidenceID{}, req.EvidenceIDs...)
		f.Status = contracts.FindingStatusNew
		if err := s.persist(f); err != nil {
			return out, fmt.Errorf("persist new finding %s: %w", f.ID, err)
		}
		if s.repo == nil {
			s.mu.Lock()
			s.items[f.ID] = f
			if s.byFingerprint == nil {
				s.byFingerprint = map[string]*domainfinding.Finding{}
			}
			s.byFingerprint[fp] = f
			s.mu.Unlock()
		}
		out = append(out, IngestResult{Finding: f, Action: contracts.FindingStatusNew})
	}
	return out, nil
}

// Resolve marks a finding resolved (does not delete history).
func (s *Service) Resolve(id contracts.FindingID) (*domainfinding.Finding, error) {
	if s == nil {
		return nil, contracts.ErrInvalidArgument
	}
	if s.repo != nil {
		f, err := s.repo.GetByID(context.Background(), id)
		if err != nil {
			return nil, err
		}
		f.Status = contracts.FindingStatusResolved
		f.LastSeen = time.Now().UTC()
		if err := s.persist(f); err != nil {
			return nil, fmt.Errorf("persist finding resolution: %w", err)
		}
		return f, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.items[id]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	f.Status = contracts.FindingStatusResolved
	f.LastSeen = time.Now().UTC()
	return f, nil
}

// GetByFingerprint returns a finding by fingerprint.
func (s *Service) GetByFingerprint(fp string) (*domainfinding.Finding, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, ok := s.byFingerprint[fp]
	if !ok {
		return nil, contracts.ErrNotFound
	}
	cp := *f
	return &cp, nil
}

// ListByProjectPage reads an authoritative project page from the durable
// repository when configured. Read errors are never converted to an empty list.
func (s *Service) ListByProjectPage(ctx context.Context, projectID contracts.ProjectID, limit, offset int) ([]*domainfinding.Finding, error) {
	if s == nil || projectID == "" || limit < 1 || limit > 500 || offset < 0 {
		return nil, contracts.ErrInvalidArgument
	}
	if s.repo != nil {
		return s.repo.ListByProject(ctx, projectID, limit, offset)
	}
	return pageFindings(s.filter(func(f *domainfinding.Finding) bool { return f.ProjectID == projectID }), limit, offset), nil
}

// ListByAssessmentPage is the durable read path for one assessment.
func (s *Service) ListByAssessmentPage(ctx context.Context, assessmentID string, limit, offset int) ([]*domainfinding.Finding, error) {
	if s == nil || assessmentID == "" || limit < 1 || limit > 500 || offset < 0 {
		return nil, contracts.ErrInvalidArgument
	}
	if s.repo != nil {
		return s.repo.ListByAssessment(ctx, assessmentID, limit, offset)
	}
	return pageFindings(s.filter(func(f *domainfinding.Finding) bool { return f.AssessmentID == assessmentID }), limit, offset), nil
}

// ListByProject returns the legacy in-memory page.
func (s *Service) ListByProject(projectID contracts.ProjectID) []*domainfinding.Finding {
	rows, _ := s.ListByProjectPage(context.Background(), projectID, 500, 0)
	return rows
}

// ListByAssessment returns the legacy in-memory page.
func (s *Service) ListByAssessment(assessmentID string) []*domainfinding.Finding {
	rows, _ := s.ListByAssessmentPage(context.Background(), assessmentID, 500, 0)
	return rows
}

func pageFindings(rows []*domainfinding.Finding, limit, offset int) []*domainfinding.Finding {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ProducedAt.Equal(rows[j].ProducedAt) {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].ProducedAt.After(rows[j].ProducedAt)
	})
	if offset >= len(rows) {
		return []*domainfinding.Finding{}
	}
	rows = rows[offset:]
	if len(rows) > limit {
		rows = rows[:limit]
	}
	return rows
}

// ListByTask returns findings for a task.
// ListByExecution returns findings for an execution id.
func (s *Service) ListByExecution(execID string) []*domainfinding.Finding {
	return s.filter(func(f *domainfinding.Finding) bool { return f.ExecutionID == execID })
}

// ListBySeverity filters by severity.
func (s *Service) ListBySeverity(sev contracts.FindingSeverity) []*domainfinding.Finding {
	return s.filter(func(f *domainfinding.Finding) bool { return f.Severity == sev })
}

func (s *Service) filter(pred func(*domainfinding.Finding) bool) []*domainfinding.Finding {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*domainfinding.Finding
	for _, f := range s.items {
		if pred(f) {
			cp := *f
			out = append(out, &cp)
		}
	}
	return out
}

func parseSeverity(s string, def contracts.FindingSeverity) contracts.FindingSeverity {
	switch contracts.FindingSeverity(s) {
	case contracts.FindingSeverityInfo, contracts.FindingSeverityLow, contracts.FindingSeverityMedium,
		contracts.FindingSeverityHigh, contracts.FindingSeverityCritical:
		return contracts.FindingSeverity(s)
	default:
		return def
	}
}

func appendUnique(base []contracts.EvidenceID, add ...contracts.EvidenceID) []contracts.EvidenceID {
	seen := map[contracts.EvidenceID]bool{}
	for _, e := range base {
		seen[e] = true
	}
	for _, e := range add {
		if !seen[e] {
			base = append(base, e)
			seen[e] = true
		}
	}
	return base
}

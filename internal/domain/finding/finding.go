// Package finding implements the Finding domain entity.
// Findings are interpreted security observations derived from Evidence.
package finding

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Finding is a structured security observation derived from one or more pieces of evidence.
type Finding struct {
	ID            contracts.FindingID
	TaskID        contracts.TaskID
	ProjectID     contracts.ProjectID
	ScopeID       contracts.ScopeID
	EvidenceIDs   []contracts.EvidenceID
	Severity      contracts.FindingSeverity
	Title         string
	Description   string
	ProducedAt    time.Time
	CorrelationID contracts.CorrelationID
	Metadata      map[string]string
}

// New creates a valid Finding or returns an error.
func New(
	id contracts.FindingID,
	taskID contracts.TaskID,
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
	severity contracts.FindingSeverity,
	title string,
) (*Finding, error) {
	if err := contracts.ValidateNonEmpty("finding.id", string(id)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("finding.task_id", string(taskID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("finding.project_id", string(projectID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("finding.scope_id", string(scopeID)); err != nil {
		return nil, err
	}
	if !severity.Valid() {
		return nil, fmt.Errorf("%w: invalid severity %q", contracts.ErrInvalidArgument, severity)
	}
	if err := contracts.ValidateNonEmpty("finding.title", title); err != nil {
		return nil, err
	}
	now := contracts.NowUTC()
	return &Finding{
		ID:          id,
		TaskID:      taskID,
		ProjectID:   projectID,
		ScopeID:     scopeID,
		Severity:    severity,
		Title:       title,
		ProducedAt:  now,
		EvidenceIDs: make([]contracts.EvidenceID, 0),
		Metadata:    make(map[string]string),
	}, nil
}

// Validate checks structural invariants.
func (f *Finding) Validate() error {
	if f == nil {
		return fmt.Errorf("%w: finding is nil", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("finding.id", string(f.ID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("finding.task_id", string(f.TaskID)); err != nil {
		return err
	}
	if !f.Severity.Valid() {
		return fmt.Errorf("%w: invalid severity %q", contracts.ErrInvalidArgument, f.Severity)
	}
	if err := contracts.ValidateNonEmpty("finding.title", f.Title); err != nil {
		return err
	}
	if f.ProducedAt.IsZero() {
		return fmt.Errorf("%w: produced_at is required", contracts.ErrInvalidArgument)
	}
	return nil
}

// AddEvidence links an evidence ID to the finding.
func (f *Finding) AddEvidence(id contracts.EvidenceID) {
	if f == nil {
		return
	}
	f.EvidenceIDs = append(f.EvidenceIDs, id)
}

// Ref returns a lightweight contract reference.
func (f *Finding) Ref() contracts.FindingRef {
	return contracts.FindingRef{ID: f.ID}
}

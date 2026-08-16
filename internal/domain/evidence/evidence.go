// Package evidence implements the Evidence domain entity.
package evidence

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Evidence is structured output produced by an execution under a grant.
type Evidence struct {
	ID           contracts.EvidenceID
	TaskID       contracts.TaskID
	GrantID      contracts.ExecutionGrantID
	CapabilityID contracts.CapabilityID
	Principal    contracts.PrincipalRef
	ProjectID    contracts.ProjectID
	ScopeID      contracts.ScopeID
	ContentType  string
	Digest       string
	// Payload is opaque, capability-specific content.
	Payload       map[string]any
	ProducedAt    time.Time
	CorrelationID contracts.CorrelationID
	Metadata      map[string]string
}

// New creates valid Evidence or returns an error.
func New(
	id contracts.EvidenceID,
	taskID contracts.TaskID,
	grantID contracts.ExecutionGrantID,
	capabilityID contracts.CapabilityID,
	principal contracts.PrincipalRef,
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
) (*Evidence, error) {
	if err := contracts.ValidateNonEmpty("evidence.id", string(id)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("evidence.task_id", string(taskID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("evidence.grant_id", string(grantID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("evidence.capability_id", string(capabilityID)); err != nil {
		return nil, err
	}
	if err := principal.Validate(); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("evidence.project_id", string(projectID)); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("evidence.scope_id", string(scopeID)); err != nil {
		return nil, err
	}
	now := contracts.NowUTC()
	return &Evidence{
		ID:           id,
		TaskID:       taskID,
		GrantID:      grantID,
		CapabilityID: capabilityID,
		Principal:    principal,
		ProjectID:    projectID,
		ScopeID:      scopeID,
		ProducedAt:   now,
		Payload:      make(map[string]any),
		Metadata:     make(map[string]string),
	}, nil
}

// Validate checks structural invariants.
func (e *Evidence) Validate() error {
	if e == nil {
		return fmt.Errorf("%w: evidence is nil", contracts.ErrInvalidArgument)
	}
	if err := contracts.ValidateNonEmpty("evidence.id", string(e.ID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("evidence.task_id", string(e.TaskID)); err != nil {
		return err
	}
	if err := contracts.ValidateNonEmpty("evidence.grant_id", string(e.GrantID)); err != nil {
		return err
	}
	if err := e.Principal.Validate(); err != nil {
		return err
	}
	if e.ProducedAt.IsZero() {
		return fmt.Errorf("%w: produced_at is required", contracts.ErrInvalidArgument)
	}
	return nil
}

// Ref returns a lightweight contract reference.
func (e *Evidence) Ref() contracts.EvidenceRef {
	return contracts.EvidenceRef{ID: e.ID}
}

// MetadataView returns the contract-level metadata view.
func (e *Evidence) MetadataView() contracts.EvidenceMetadata {
	return contracts.EvidenceMetadata{
		ID:            e.ID,
		TaskID:        e.TaskID,
		GrantID:       e.GrantID,
		CapabilityID:  e.CapabilityID,
		Principal:     e.Principal,
		ProducedAt:    e.ProducedAt,
		ContentType:   e.ContentType,
		Digest:        e.Digest,
		CorrelationID: e.CorrelationID,
	}
}

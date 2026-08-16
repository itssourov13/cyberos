// Package provenance implements the Provenance domain entity.
// Provenance answers WHO / WHAT / WHEN / WHERE / WHY and under which
// scope / task / capability / grant the subject was produced.
package provenance

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Provenance is the full lineage record for Evidence or Findings.
type Provenance struct {
	contracts.Provenance
}

// NewFromContract constructs domain Provenance after validation.
func NewFromContract(p contracts.Provenance) (*Provenance, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &Provenance{Provenance: p}, nil
}

// NewForEvidence is a convenience constructor for evidence provenance.
func NewForEvidence(
	evidenceID contracts.EvidenceID,
	principal contracts.PrincipalRef,
	projectID contracts.ProjectID,
	scopeID contracts.ScopeID,
	taskID contracts.TaskID,
	capabilityID contracts.CapabilityID,
	grantID contracts.ExecutionGrantID,
	executorID string,
) (*Provenance, error) {
	p := contracts.Provenance{
		SubjectType:  "evidence",
		SubjectID:    string(evidenceID),
		Principal:    principal,
		ProjectID:    projectID,
		ScopeID:      scopeID,
		TaskID:       taskID,
		CapabilityID: capabilityID,
		GrantID:      grantID,
		ExecutorID:   executorID,
		ProducedAt:   contracts.NowUTC(),
	}
	return NewFromContract(p)
}

// Validate delegates to the contract.
func (p *Provenance) Validate() error {
	if p == nil {
		return fmt.Errorf("%w: provenance is nil", contracts.ErrInvalidArgument)
	}
	return p.Provenance.Validate()
}

// ProducedAtTime returns the production timestamp.
func (p *Provenance) ProducedAtTime() time.Time {
	if p == nil {
		return time.Time{}
	}
	return p.ProducedAt
}

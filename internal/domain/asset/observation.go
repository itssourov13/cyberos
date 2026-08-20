package asset

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// Observation is a normalized tool observation bound to an asset (not a Finding).
type Observation struct {
	ID            string
	AssetID       contracts.AssetID
	ProjectID     contracts.ProjectID
	ScopeID       contracts.ScopeID
	TaskID        contracts.TaskID
	ExecutionID   string
	ToolID        string
	CapabilityID  contracts.CapabilityID
	Kind          contracts.ObservationKind
	Summary       string
	Detail        string
	EndpointID    string
	EvidenceIDs   []contracts.EvidenceID
	CorrelationID contracts.CorrelationID
	ObservedAt    time.Time
	Metadata      map[string]string
}

// NewObservation creates an observation.
func NewObservation(id string, assetID contracts.AssetID, projectID contracts.ProjectID, kind contracts.ObservationKind, summary string) (*Observation, error) {
	if err := contracts.ValidateNonEmpty("observation.id", id); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("observation.asset_id", string(assetID)); err != nil {
		return nil, err
	}
	if !kind.Valid() {
		return nil, fmt.Errorf("%w: invalid observation kind", contracts.ErrInvalidArgument)
	}
	return &Observation{
		ID: id, AssetID: assetID, ProjectID: projectID, Kind: kind, Summary: summary,
		ObservedAt: contracts.NowUTC(), EvidenceIDs: nil, Metadata: map[string]string{},
	}, nil
}

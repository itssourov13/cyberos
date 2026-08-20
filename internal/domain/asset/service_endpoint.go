package asset

import (
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// ServiceEndpoint is a discovered service/port/URL on an asset.
type ServiceEndpoint struct {
	ID        string
	AssetID   contracts.AssetID
	ProjectID contracts.ProjectID
	Protocol  string // tcp, udp, http, https
	Port      int
	Path      string
	Banner    string
	FirstSeen time.Time
	LastSeen  time.Time
	Metadata  map[string]string
}

// NewServiceEndpoint creates a service endpoint record.
func NewServiceEndpoint(id string, assetID contracts.AssetID, projectID contracts.ProjectID, protocol string, port int) (*ServiceEndpoint, error) {
	if err := contracts.ValidateNonEmpty("endpoint.id", id); err != nil {
		return nil, err
	}
	if err := contracts.ValidateNonEmpty("endpoint.asset_id", string(assetID)); err != nil {
		return nil, err
	}
	now := contracts.NowUTC()
	return &ServiceEndpoint{
		ID: id, AssetID: assetID, ProjectID: projectID,
		Protocol: protocol, Port: port,
		FirstSeen: now, LastSeen: now, Metadata: map[string]string{},
	}, nil
}

// Validate checks required fields.
func (e *ServiceEndpoint) Validate() error {
	if e == nil {
		return fmt.Errorf("%w: endpoint nil", contracts.ErrInvalidArgument)
	}
	return contracts.ValidateNonEmpty("endpoint.id", e.ID)
}

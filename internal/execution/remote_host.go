package execution

import (
	"context"
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// RemoteHost is registered for future remote workers (Phase 17D).
// Phase 17C: Execute always fails closed — identity only, no remote job dispatch.
type RemoteHost struct {
	HostID HostID
}

func NewRemoteHost(id HostID) (*RemoteHost, error) {
	if err := ValidateHostID(id); err != nil {
		return nil, err
	}
	if id == HostLocal || id == HostContainer {
		return nil, fmt.Errorf("%w: reserved host id", contracts.ErrInvalidArgument)
	}
	return &RemoteHost{HostID: id}, nil
}

func (h *RemoteHost) ID() HostID {
	if h == nil || h.HostID == "" {
		return "remote"
	}
	return h.HostID
}

func (h *RemoteHost) Execute(ctx context.Context, req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	_ = ctx
	_ = req
	now := time.Now().UTC()
	return contracts.ExecutionResult{
		Status: contracts.ExecutionStatusRejected, StartedAt: now, FinishedAt: now,
		ErrorMessage: "remote execution not enabled (Phase 17C identity only)",
		ExecutorID:   "remote", HostID: string(h.ID()),
	}, fmt.Errorf("%w: remote execution not implemented until Phase 17D", contracts.ErrInvalidArgument)
}

var _ ExecutionHost = (*RemoteHost)(nil)

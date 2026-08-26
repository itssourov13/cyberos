package execution

import (
	"context"
	"fmt"

	"cyberos/internal/contracts"
)

// LocalHost adapts the existing in-process Executor/Router/Sandbox path
// as an ExecutionHost. It does not authorize, issue Grants, or call CP.
type LocalHost struct {
	Executor contracts.Executor
}

// NewLocalHost wraps an existing contracts.Executor (typically a Router).
func NewLocalHost(ex contracts.Executor) *LocalHost {
	return &LocalHost{Executor: ex}
}

func (h *LocalHost) ID() HostID { return HostLocal }

// Execute dispatches to the underlying executor. Authorization is the Broker's job.
func (h *LocalHost) Execute(ctx context.Context, req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	_ = ctx // cancellation reserved for future host implementations
	if h == nil || h.Executor == nil {
		return contracts.ExecutionResult{}, fmt.Errorf("%w: local host not configured", contracts.ErrInvalidArgument)
	}
	return h.Executor.Execute(req)
}

var _ ExecutionHost = (*LocalHost)(nil)

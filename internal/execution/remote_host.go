package execution

import (
	"context"
	"fmt"
	"time"

	"cyberos/internal/contracts"
)

// RemoteHost dispatches authorized work to a registered ACTIVE worker via
// durable assignments. Grant validation remains in Broker before this host runs.
type RemoteHost struct {
	HostID      HostID
	Workers     *WorkerRegistry
	Assignments *AssignmentService
	LeaseTTL    time.Duration
	// AutoComplete simulates an in-process worker for tests (optional).
	AutoComplete bool
}

// NewRemoteHost constructs a remote host bound to a logical HostID.
func NewRemoteHost(id HostID, workers *WorkerRegistry, assignments *AssignmentService) (*RemoteHost, error) {
	if err := ValidateHostID(id); err != nil {
		return nil, err
	}
	if id == HostLocal || id == HostContainer {
		return nil, fmt.Errorf("%w: reserved host id", contracts.ErrInvalidArgument)
	}
	if workers == nil || assignments == nil {
		return nil, fmt.Errorf("%w: workers and assignments required", contracts.ErrInvalidArgument)
	}
	return &RemoteHost{
		HostID: id, Workers: workers, Assignments: assignments,
		LeaseTTL: 2 * time.Minute,
	}, nil
}

func (h *RemoteHost) ID() HostID {
	if h == nil || h.HostID == "" {
		return "remote"
	}
	return h.HostID
}

// Execute creates/dispatches an assignment for the bound worker.
// Requires Parameters["worker_id"] selected server-side by the caller after Grant validation.
func (h *RemoteHost) Execute(ctx context.Context, req contracts.ExecutionRequest) (contracts.ExecutionResult, error) {
	start := time.Now().UTC()
	if h == nil || h.Workers == nil || h.Assignments == nil {
		return remoteFail(start, "remote host not configured"), fmt.Errorf("%w: remote host not configured", contracts.ErrInvalidArgument)
	}

	projectID := req.Grant.ProjectID
	workerID := ""
	if req.Parameters != nil {
		if v, ok := req.Parameters["worker_id"].(string); ok {
			workerID = v
		}
	}
	if workerID == "" {
		return remoteFail(start, "worker_id required"), fmt.Errorf("%w: worker_id required", contracts.ErrInvalidArgument)
	}

	// Authenticate worker identity (ACTIVE, not revoked/expired, project-scoped).
	worker, err := h.Workers.Get(ctx, projectID, workerID)
	if err != nil {
		return remoteFail(start, "worker not found"), err
	}
	if worker.HostID != h.HostID {
		return remoteFail(start, "host mismatch"), fmt.Errorf("%w: worker host mismatch", contracts.ErrForbidden)
	}
	if err := worker.CanAuthenticate(time.Now().UTC()); err != nil {
		return remoteFail(start, err.Error()), err
	}
	if worker.Status != WorkerActive {
		return remoteFail(start, "worker not active"), fmt.Errorf("%w: worker not active", contracts.ErrForbidden)
	}

	execID := string(req.Grant.ID)
	if req.Parameters != nil {
		if v, ok := req.Parameters["execution_id"].(string); ok && v != "" {
			execID = v
		}
	}
	assignID := "asg-" + execID
	if existing, err := h.Assignments.Get(ctx, projectID, assignID); err == nil {
		// Idempotent: return current state without re-running
		return assignmentToResult(existing, start, req.CorrelationID), nil
	}

	a, err := h.Assignments.Create(ctx, ExecutionAssignment{
		ID: assignID, ExecutionID: execID, ProjectID: projectID, TenantID: "",
		HostID: h.HostID, WorkerID: workerID, WorkflowRunID: string(req.Grant.TaskID),
	})
	if err != nil {
		return remoteFail(start, err.Error()), err
	}
	a, err = h.Assignments.Dispatch(ctx, projectID, a.ID)
	if err != nil {
		return remoteFail(start, err.Error()), err
	}

	if h.AutoComplete {
		lease := h.LeaseTTL
		if lease <= 0 {
			lease = 2 * time.Minute
		}
		a, err = h.Assignments.Claim(ctx, projectID, a.ID, workerID, lease)
		if err != nil {
			return remoteFail(start, err.Error()), err
		}
		hash := HashResult("SUCCEEDED", "remote-ok")
		a, err = h.Assignments.Complete(ctx, projectID, a.ID, workerID, hash, "SUCCEEDED", "")
		if err != nil {
			return remoteFail(start, err.Error()), err
		}
	}

	return assignmentToResult(a, start, req.CorrelationID), nil
}

func assignmentToResult(a ExecutionAssignment, start time.Time, corr contracts.CorrelationID) contracts.ExecutionResult {
	finished := time.Now().UTC()
	status := contracts.ExecutionStatusSucceeded
	switch a.Status {
	case AssignFailed:
		status = contracts.ExecutionStatusFailed
	case AssignCancelled:
		status = contracts.ExecutionStatusCancelled
	case AssignExpired:
		status = contracts.ExecutionStatusTimeout
	case AssignPending, AssignDispatched, AssignClaimed, AssignRunning:
		status = contracts.ExecutionStatusSucceeded // dispatch accepted; async completion in full protocol
	}
	return contracts.ExecutionResult{
		Status: status, StartedAt: start, FinishedAt: finished,
		Output: map[string]any{
			"assignment_id": a.ID, "execution_id": a.ExecutionID, "worker_id": a.WorkerID,
			"assignment_status": string(a.Status), "result_hash": a.ResultHash,
		},
		ErrorMessage: a.ErrorMessage, ExecutorID: "remote", HostID: string(a.HostID),
		CorrelationID: corr,
	}
}

func remoteFail(start time.Time, msg string) contracts.ExecutionResult {
	now := time.Now().UTC()
	return contracts.ExecutionResult{
		Status: contracts.ExecutionStatusRejected, StartedAt: start, FinishedAt: now,
		ErrorMessage: msg, ExecutorID: "remote", HostID: "remote",
	}
}

var _ ExecutionHost = (*RemoteHost)(nil)

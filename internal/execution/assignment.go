package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// AssignmentStatus is the lifecycle of a remote execution assignment.
type AssignmentStatus string

const (
	AssignPending    AssignmentStatus = "PENDING"
	AssignDispatched AssignmentStatus = "DISPATCHED"
	AssignClaimed    AssignmentStatus = "CLAIMED"
	AssignRunning    AssignmentStatus = "RUNNING"
	AssignCompleted  AssignmentStatus = "COMPLETED"
	AssignFailed     AssignmentStatus = "FAILED"
	AssignCancelled  AssignmentStatus = "CANCELLED"
	AssignExpired    AssignmentStatus = "EXPIRED"
)

// ExecutionAssignment is durable controller-owned work for a remote worker.
// No Grant fields are stored.
type ExecutionAssignment struct {
	ID                string
	ExecutionID       string
	WorkflowRunID     string
	ProjectID         contracts.ProjectID
	TenantID          contracts.TenantID
	HostID            HostID
	WorkerID          string
	GrantID           string
	PackContentDigest string
	Status            AssignmentStatus
	Attempt           int
	MaxAttempts       int
	LeaseUntil        time.Time
	CreatedAt         time.Time
	DispatchedAt      time.Time
	StartedAt         time.Time
	CompletedAt       time.Time
	CancelledAt       time.Time
	ResultHash        string
	ResultStatus      string
	ErrorMessage      string
	Version           int64
}

func (a ExecutionAssignment) IsTerminal() bool {
	switch a.Status {
	case AssignCompleted, AssignFailed, AssignCancelled, AssignExpired:
		return true
	default:
		return false
	}
}

// AssignmentService manages create/claim/complete/cancel with CAS semantics.
type AssignmentService struct {
	mu   sync.Mutex
	repo ports.AssignmentRepository
	now  func() time.Time
}

func NewAssignmentService(repo ports.AssignmentRepository) *AssignmentService {
	return &AssignmentService{repo: repo, now: func() time.Time { return time.Now().UTC() }}
}

// SetNow overrides the clock for tests (Phase 22C).
func (s *AssignmentService) SetNow(fn func() time.Time) {
	if s != nil && fn != nil {
		s.now = fn
	}
}

// CreateFromGrant binds Grant identity into a new PENDING assignment (Phase 22B).
func (s *AssignmentService) CreateFromGrant(ctx context.Context, g contracts.ExecutionGrant, host HostID, workerID, assignmentID string) (ExecutionAssignment, error) {
	if err := g.Validate(); err != nil {
		return ExecutionAssignment{}, err
	}
	a := ExecutionAssignment{
		ID: assignmentID, ExecutionID: string(g.ID), ProjectID: g.ProjectID,
		HostID: host, WorkerID: workerID, GrantID: string(g.ID),
		PackContentDigest: g.PackContentDigest,
	}
	return s.Create(ctx, a)
}

// Create persists a PENDING assignment (controller-side, after Grant validation).
func (s *AssignmentService) Create(ctx context.Context, a ExecutionAssignment) (ExecutionAssignment, error) {
	if s == nil || s.repo == nil {
		return ExecutionAssignment{}, fmt.Errorf("%w: assignment service not configured", contracts.ErrInvalidArgument)
	}
	if a.ID == "" || a.ExecutionID == "" || a.ProjectID == "" || a.WorkerID == "" {
		return ExecutionAssignment{}, fmt.Errorf("%w: assignment identity incomplete", contracts.ErrInvalidArgument)
	}
	if err := ValidateHostID(a.HostID); err != nil {
		return ExecutionAssignment{}, err
	}
	now := s.now()
	a.Status = AssignPending
	a.CreatedAt = now
	a.Version = 1
	if a.MaxAttempts <= 0 {
		a.MaxAttempts = 1
	}
	if err := s.repo.Create(ctx, assignmentToRecord(a)); err != nil {
		return ExecutionAssignment{}, err
	}
	return a, nil
}

// Dispatch marks PENDING → DISPATCHED.
func (s *AssignmentService) Dispatch(ctx context.Context, projectID contracts.ProjectID, id string) (ExecutionAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.get(ctx, projectID, id)
	if err != nil {
		return ExecutionAssignment{}, err
	}
	if a.Status != AssignPending && a.Status != AssignDispatched {
		return ExecutionAssignment{}, fmt.Errorf("%w: cannot dispatch from %s", contracts.ErrConflict, a.Status)
	}
	now := s.now()
	a.Status = AssignDispatched
	a.DispatchedAt = now
	a.Attempt++
	if err := s.repo.Update(ctx, assignmentToRecord(a)); err != nil {
		return ExecutionAssignment{}, err
	}
	return a, nil
}

// Claim atomically claims DISPATCHED/expired lease for workerID (Phase 20: ClaimAtomic).
func (s *AssignmentService) Claim(ctx context.Context, projectID contracts.ProjectID, id, workerID string, lease time.Duration) (ExecutionAssignment, error) {
	now := s.now()
	leaseUntil := now.Add(lease)
	rec, err := s.repo.ClaimAtomic(ctx, projectID, id, workerID, leaseUntil)
	if err != nil {
		return ExecutionAssignment{}, err
	}
	return recordToAssignment(rec), nil
}

// RenewLease extends ownership.
func (s *AssignmentService) RenewLease(ctx context.Context, projectID contracts.ProjectID, id, workerID string, lease time.Duration) (ExecutionAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.get(ctx, projectID, id)
	if err != nil {
		return ExecutionAssignment{}, err
	}
	if a.WorkerID != workerID {
		return ExecutionAssignment{}, fmt.Errorf("%w: worker mismatch", contracts.ErrForbidden)
	}
	if a.Status != AssignClaimed && a.Status != AssignRunning {
		return ExecutionAssignment{}, fmt.Errorf("%w: not leasable", contracts.ErrConflict)
	}
	now := s.now()
	if !a.LeaseUntil.IsZero() && now.After(a.LeaseUntil) {
		return ExecutionAssignment{}, fmt.Errorf("%w: lease expired", contracts.ErrConflict)
	}
	a.Status = AssignRunning
	a.LeaseUntil = now.Add(lease)
	if err := s.repo.Update(ctx, assignmentToRecord(a)); err != nil {
		return ExecutionAssignment{}, err
	}
	return a, nil
}

// Complete accepts an authenticated worker result (idempotent on same hash).
// Phase 22C: expired lease and terminal states fail closed.
func (s *AssignmentService) Complete(ctx context.Context, projectID contracts.ProjectID, id, workerID, resultHash, resultStatus, errMsg string) (ExecutionAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.get(ctx, projectID, id)
	if err != nil {
		return ExecutionAssignment{}, err
	}
	if a.WorkerID != workerID {
		return ExecutionAssignment{}, fmt.Errorf("%w: worker mismatch", contracts.ErrForbidden)
	}
	if a.Status == AssignCompleted && a.ResultHash == resultHash {
		return a, nil // idempotent
	}
	if a.IsTerminal() {
		if a.Status == AssignCompleted && a.ResultHash != resultHash {
			return ExecutionAssignment{}, fmt.Errorf("%w: conflicting result", contracts.ErrConflict)
		}
		return ExecutionAssignment{}, fmt.Errorf("%w: assignment terminal", contracts.ErrConflict)
	}
	now := s.now()
	if !a.LeaseUntil.IsZero() && now.After(a.LeaseUntil) {
		return ExecutionAssignment{}, fmt.Errorf("%w: lease expired", contracts.ErrConflict)
	}
	if a.Status != AssignClaimed && a.Status != AssignRunning && a.Status != AssignDispatched {
		return ExecutionAssignment{}, fmt.Errorf("%w: cannot complete from %s", contracts.ErrConflict, a.Status)
	}
	a.Status = AssignCompleted
	if resultStatus == "FAILED" {
		a.Status = AssignFailed
	}
	a.CompletedAt = now
	a.ResultHash = resultHash
	a.ResultStatus = resultStatus
	a.ErrorMessage = errMsg
	if err := s.repo.Update(ctx, assignmentToRecord(a)); err != nil {
		return ExecutionAssignment{}, err
	}
	return a, nil
}

// Cancel is durable and idempotent.
func (s *AssignmentService) Cancel(ctx context.Context, projectID contracts.ProjectID, id string) (ExecutionAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.get(ctx, projectID, id)
	if err != nil {
		return ExecutionAssignment{}, err
	}
	if a.Status == AssignCancelled {
		return a, nil
	}
	if a.Status == AssignCompleted || a.Status == AssignFailed {
		return a, nil // completion already won
	}
	now := s.now()
	a.Status = AssignCancelled
	a.CancelledAt = now
	a.CompletedAt = now
	if err := s.repo.Update(ctx, assignmentToRecord(a)); err != nil {
		return ExecutionAssignment{}, err
	}
	return a, nil
}

// RecoverExpired marks lease-expired non-terminal assignments EXPIRED.
func (s *AssignmentService) RecoverExpired(ctx context.Context, projectID contracts.ProjectID, id string) (ExecutionAssignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	a, err := s.get(ctx, projectID, id)
	if err != nil {
		return ExecutionAssignment{}, err
	}
	if a.IsTerminal() {
		return a, nil
	}
	now := s.now()
	if a.LeaseUntil.IsZero() || !now.After(a.LeaseUntil) {
		return ExecutionAssignment{}, fmt.Errorf("%w: lease still valid", contracts.ErrConflict)
	}
	a.Status = AssignExpired
	a.CompletedAt = now
	if err := s.repo.Update(ctx, assignmentToRecord(a)); err != nil {
		return ExecutionAssignment{}, err
	}
	return a, nil
}

func (s *AssignmentService) Get(ctx context.Context, projectID contracts.ProjectID, id string) (ExecutionAssignment, error) {
	return s.get(ctx, projectID, id)
}

// List returns a project's assignments, newest first (read-only).
func (s *AssignmentService) List(ctx context.Context, projectID contracts.ProjectID, limit int) ([]ExecutionAssignment, error) {
	recs, err := s.repo.ListByProject(ctx, projectID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]ExecutionAssignment, 0, len(recs))
	for _, r := range recs {
		out = append(out, recordToAssignment(r))
	}
	return out, nil
}

func (s *AssignmentService) get(ctx context.Context, projectID contracts.ProjectID, id string) (ExecutionAssignment, error) {
	rec, err := s.repo.Get(ctx, projectID, id)
	if err != nil {
		return ExecutionAssignment{}, err
	}
	return recordToAssignment(rec), nil
}

// HashResult produces a stable result integrity hash.
func HashResult(status, output string) string {
	h := sha256.Sum256([]byte(status + "\n" + output))
	return hex.EncodeToString(h[:])
}

func assignmentToRecord(a ExecutionAssignment) ports.AssignmentRecord {
	return ports.AssignmentRecord{
		ID: a.ID, ExecutionID: a.ExecutionID, WorkflowRunID: a.WorkflowRunID,
		ProjectID: a.ProjectID, TenantID: a.TenantID, HostID: string(a.HostID), WorkerID: a.WorkerID,
		GrantID: a.GrantID, PackContentDigest: a.PackContentDigest,
		Status: string(a.Status), Attempt: a.Attempt, MaxAttempts: a.MaxAttempts,
		LeaseUntil: a.LeaseUntil, CreatedAt: a.CreatedAt, DispatchedAt: a.DispatchedAt,
		StartedAt: a.StartedAt, CompletedAt: a.CompletedAt, CancelledAt: a.CancelledAt,
		ResultHash: a.ResultHash, ResultStatus: a.ResultStatus, ErrorMessage: a.ErrorMessage, Version: a.Version,
	}
}

func recordToAssignment(r ports.AssignmentRecord) ExecutionAssignment {
	return ExecutionAssignment{
		ID: r.ID, ExecutionID: r.ExecutionID, WorkflowRunID: r.WorkflowRunID,
		ProjectID: r.ProjectID, TenantID: r.TenantID, HostID: HostID(r.HostID), WorkerID: r.WorkerID,
		GrantID: r.GrantID, PackContentDigest: r.PackContentDigest,
		Status: AssignmentStatus(r.Status), Attempt: r.Attempt, MaxAttempts: r.MaxAttempts,
		LeaseUntil: r.LeaseUntil, CreatedAt: r.CreatedAt, DispatchedAt: r.DispatchedAt,
		StartedAt: r.StartedAt, CompletedAt: r.CompletedAt, CancelledAt: r.CancelledAt,
		ResultHash: r.ResultHash, ResultStatus: r.ResultStatus, ErrorMessage: r.ErrorMessage, Version: r.Version,
	}
}

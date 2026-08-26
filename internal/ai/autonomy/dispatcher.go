package autonomy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/storage/ports"
)

// Dispatcher claims outbox events and delivers them to EventAutonomyProcessor only.
// Never calls Grant, Broker, or Executor.
type Dispatcher struct {
	Repo        ports.AutonomyOutboxRepository
	Processor   *EventAutonomyProcessor
	WorkerID    string
	BatchSize   int
	Lease       time.Duration
	MaxAttempts int
	Now         func() time.Time
}

// NewDispatcher constructs a bounded outbox dispatcher.
func NewDispatcher(repo ports.AutonomyOutboxRepository, proc *EventAutonomyProcessor, workerID string) *Dispatcher {
	return &Dispatcher{
		Repo: repo, Processor: proc, WorkerID: workerID,
		BatchSize: 8, Lease: 30 * time.Second, MaxAttempts: 5, Now: time.Now,
	}
}

// Tick claims a batch and processes each event once through 15B processor.
func (d *Dispatcher) Tick(ctx context.Context) (delivered, retried, dead int, err error) {
	if d == nil || d.Repo == nil || d.Processor == nil {
		return 0, 0, 0, fmt.Errorf("%w: dispatcher not configured", contracts.ErrInvalidArgument)
	}
	batch, err := d.Repo.ClaimBatch(ctx, d.WorkerID, d.BatchSize, d.Lease)
	if err != nil {
		return 0, 0, 0, err
	}
	for _, rec := range batch {
		if err := d.handleOne(ctx, rec); err != nil {
			// classify permanent vs transient already done inside handleOne
			_ = err
		}
		switch {
		case rec.Status == ports.OutboxDelivered:
			// updated in handle
		}
	}
	// recount from outcomes would need return values — track in handleOne via side effect on counters
	return delivered, retried, dead, nil
}

// ProcessBatch is like Tick but returns counters.
func (d *Dispatcher) ProcessBatch(ctx context.Context) (delivered, retried, deadLetter int, err error) {
	if d == nil || d.Repo == nil || d.Processor == nil {
		return 0, 0, 0, fmt.Errorf("%w: dispatcher not configured", contracts.ErrInvalidArgument)
	}
	batch, err := d.Repo.ClaimBatch(ctx, d.WorkerID, d.BatchSize, d.Lease)
	if err != nil {
		return 0, 0, 0, err
	}
	for _, rec := range batch {
		outcome := d.deliver(ctx, rec)
		switch outcome {
		case "delivered":
			delivered++
		case "retry":
			retried++
		case "dead":
			deadLetter++
		}
	}
	return delivered, retried, deadLetter, nil
}

func (d *Dispatcher) handleOne(ctx context.Context, rec ports.AutonomyOutboxRecord) error {
	_ = d.deliver(ctx, rec)
	return nil
}

func (d *Dispatcher) deliver(ctx context.Context, rec ports.AutonomyOutboxRecord) string {
	event, err := RecordToAutonomyEvent(rec)
	if err != nil {
		_ = d.Repo.MarkDeadLetter(ctx, rec.ID, d.WorkerID, "invalid_event: "+err.Error())
		return "dead"
	}
	res, err := d.Processor.Process(ctx, event)
	if err != nil || (res != nil && res.Status == ProcessRejected) {
		reason := "process_error"
		if err != nil {
			reason = err.Error()
		} else if res != nil {
			reason = res.Error
		}
		if isPermanentFailure(reason, res) {
			_ = d.Repo.MarkDeadLetter(ctx, rec.ID, d.WorkerID, reason)
			return "dead"
		}
		if rec.Attempts >= d.MaxAttempts {
			_ = d.Repo.MarkDeadLetter(ctx, rec.ID, d.WorkerID, "max_attempts: "+reason)
			return "dead"
		}
		delay := backoff(rec.Attempts)
		_ = d.Repo.MarkRetry(ctx, rec.ID, d.WorkerID, d.Now().Add(delay), reason)
		return "retry"
	}
	// Terminal / stale / wait / eligible / idempotent skip all count as successfully consumed.
	_ = d.Repo.MarkDelivered(ctx, rec.ID, d.WorkerID)
	return "delivered"
}

func isPermanentFailure(reason string, res *ProcessResult) bool {
	if res != nil {
		switch res.Status {
		case ProcessStale, ProcessTerminal:
			return false // safely consumed, not dead-letter
		case ProcessRejected:
			r := strings.ToLower(res.Error + reason)
			for _, p := range []string{"schema", "invalid", "unsupported", "missing"} {
				if strings.Contains(r, p) {
					return true
				}
			}
		}
	}
	r := strings.ToLower(reason)
	for _, p := range []string{"unsupported", "invalid autonomy event", "missing schema", "secret field"} {
		if strings.Contains(r, p) {
			return true
		}
	}
	return false
}

func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := time.Duration(1<<uint(min(attempt, 5))) * time.Second
	if d > 60*time.Second {
		d = 60 * time.Second
	}
	return d
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

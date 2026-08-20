package runner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cyberos/internal/contracts"
	"cyberos/internal/events"
)

// ProcessFunc is invoked after a successful claim.
// Implementations that execute work MUST still obtain a Control Plane decision
// and ExecutionGrant; this callback is never an authorization boundary.
type ProcessFunc func(ctx context.Context, claim Claim) error

// Worker runs a durable claim → process → release loop.
type Worker struct {
	Runner   *Runner
	ID       WorkerID
	Batch    int
	Interval time.Duration
	Process  ProcessFunc
	Events   *events.Store
	// OnEvent is optional fan-out (e.g. SSE hub).
	OnEvent func(contracts.EventEnvelope)

	stop chan struct{}
	wg   sync.WaitGroup
}

// NewWorker constructs a worker. Process may be nil (claim+release only, no execution).
func NewWorker(r *Runner, id WorkerID, process ProcessFunc) *Worker {
	return &Worker{
		Runner:   r,
		ID:       id,
		Batch:    5,
		Interval: 200 * time.Millisecond,
		Process:  process,
		stop:     make(chan struct{}),
	}
}

// TickOnce runs one promote + claim + process + release cycle (for tests).
func (w *Worker) TickOnce(ctx context.Context) (claimed int, err error) {
	if w == nil || w.Runner == nil {
		return 0, fmt.Errorf("%w: worker not configured", contracts.ErrInvalidArgument)
	}
	promoted, err := w.Runner.PromoteDueScheduled(ctx, w.Batch)
	if err != nil {
		return 0, err
	}
	for _, tk := range promoted {
		w.emit(events.TypeRunnerPromoted, Claim{TaskID: tk.ID, WorkerID: w.ID, Task: tk}, map[string]any{
			"task_id": string(tk.ID),
			"state":   string(tk.State),
		})
	}
	claims, err := w.Runner.ClaimPending(ctx, w.ID, w.Batch)
	if err != nil {
		return 0, err
	}
	for _, c := range claims {
		w.emit(events.TypeRunnerClaimed, c, map[string]any{
			"worker_id": string(c.WorkerID),
			"task_id":   string(c.TaskID),
			"state":     string(c.Task.State),
		})
		if w.Process != nil {
			if perr := w.Process(ctx, c); perr != nil {
				w.Runner.Release(c.TaskID)
				w.emit(events.TypeRunnerReleased, c, map[string]any{
					"worker_id": string(c.WorkerID),
					"task_id":   string(c.TaskID),
					"error":     perr.Error(),
				})
				continue
			}
		}
		w.Runner.Release(c.TaskID)
		w.emit(events.TypeRunnerReleased, c, map[string]any{
			"worker_id": string(c.WorkerID),
			"task_id":   string(c.TaskID),
			"ok":        true,
		})
	}
	return len(claims), nil
}

// RecoverOnce clears expired leases and emits recovery events.
func (w *Worker) RecoverOnce() int {
	n := w.Runner.RecoverExpiredLeases()
	if n > 0 {
		w.emit(events.TypeRunnerRecovered, Claim{WorkerID: w.ID}, map[string]any{
			"recovered": n,
			"worker_id": string(w.ID),
		})
	}
	return n
}

// Start begins the background loop until Stop.
func (w *Worker) Start(ctx context.Context) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.RecoverOnce()
				_, _ = w.TickOnce(ctx)
			}
		}
	}()
}

// Stop signals the loop to exit and waits.
func (w *Worker) Stop() {
	select {
	case <-w.stop:
	default:
		close(w.stop)
	}
	w.wg.Wait()
}

func (w *Worker) emit(typ contracts.EventType, c Claim, payload map[string]any) {
	if w.Events == nil {
		if w.OnEvent != nil {
			// still notify with minimal envelope
			env := contracts.EventEnvelope{
				ID:            contracts.EventID(fmt.Sprintf("evt-%s-%d", typ, time.Now().UnixNano())),
				Type:          typ,
				SchemaVersion: contracts.EventSchemaV1,
				Timestamp:     time.Now().UTC(),
				TaskID:        c.TaskID,
				CorrelationID: contracts.CorrelationID("runner-" + string(w.ID)),
				Payload:       payload,
			}
			w.OnEvent(env)
		}
		return
	}
	env, err := w.Events.Append(events.AppendRequest{
		ID:            contracts.EventID(fmt.Sprintf("evt-%s-%d", typ, time.Now().UnixNano())),
		Type:          typ,
		TaskID:        c.TaskID,
		CorrelationID: contracts.CorrelationID("runner-" + string(w.ID)),
		Payload:       payload,
	})
	if err != nil {
		return
	}
	if w.OnEvent != nil {
		w.OnEvent(env)
	}
}

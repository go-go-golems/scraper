package workflowv3runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
)

// AttemptExecutionError means an attempt outcome was durably recorded. A
// dispatcher may continue unrelated work; deterministic RunOne callers still
// receive the underlying error.
type AttemptExecutionError struct {
	Err error
}

func (e *AttemptExecutionError) Error() string {
	return e.Err.Error()
}

func (e *AttemptExecutionError) Unwrap() error {
	return e.Err
}

type Dispatcher struct {
	Engine       *Engine
	Capacities   map[string]int
	PollInterval time.Duration
	OnStarted    func(workflowv3.Lease)
}

type dispatchCompletion struct {
	lease workflowv3.Lease
	err   error
}

// DispatchOnce performs exactly one transactional lease attempt and no task
// execution. It is the deterministic single-action test hook.
func (d *Dispatcher) DispatchOnce(ctx context.Context) (*workflowv3.Lease, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}
	return d.Engine.Store.LeaseNextWithResources(
		ctx,
		d.Engine.Registry,
		d.Capacities,
		d.Engine.now(),
		d.Engine.leaseDuration(),
	)
}

// Run continuously fills every compatible free resource slot. Completion of
// one class wakes refill immediately and never waits for unrelated attempts.
func (d *Dispatcher) Run(ctx context.Context) error {
	if err := d.validate(); err != nil {
		return err
	}
	buffer := 0
	for _, capacity := range d.Capacities {
		buffer += capacity
	}
	completions := make(chan dispatchCompletion, buffer)
	poll := time.NewTicker(d.pollInterval())
	defer poll.Stop()

	for {
		started := false
		for {
			expanded, err := d.Engine.ExpandOne(ctx)
			if err != nil {
				return fmt.Errorf("expand workflow map: %w", err)
			}
			if expanded {
				started = true
			}
			finalized, err := d.Engine.FinalizeOneMap(ctx)
			if err != nil {
				return fmt.Errorf("finalize workflow map: %w", err)
			}
			if finalized {
				started = true
			}
			reduced, err := d.Engine.ReduceOne(ctx)
			if err != nil {
				return fmt.Errorf("advance workflow reduction: %w", err)
			}
			if reduced {
				started = true
			}
			lease, err := d.DispatchOnce(ctx)
			if err != nil {
				if contextErr := ctx.Err(); contextErr != nil {
					return contextErr
				}
				return err
			}
			if lease == nil {
				break
			}
			started = true
			if d.OnStarted != nil {
				d.OnStarted(*lease)
			}
			go func(leased workflowv3.Lease) {
				err := d.Engine.ExecuteLease(ctx, leased)
				completions <- dispatchCompletion{lease: leased, err: err}
			}(*lease)
		}
		if started {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case completion := <-completions:
			if completion.err != nil && !isRecordedAttemptError(completion.err) {
				return fmt.Errorf(
					"execute %s/%s: %w",
					completion.lease.RunID,
					completion.lease.NodeKey,
					completion.err,
				)
			}
		case <-poll.C:
			// Retry deadlines, lease expiry, new runs, and cross-process
			// completions are coalesced by this wakeup.
		}
	}
}

func (d *Dispatcher) OperationalSnapshot(
	ctx context.Context,
	runID *workflowv3.RunID,
) (workflowv3.OperationalSnapshot, error) {
	if err := d.validate(); err != nil {
		return workflowv3.OperationalSnapshot{}, err
	}
	snapshot, err := d.Engine.Store.OperationalSnapshot(
		ctx, runID, d.Engine.Registry, d.Capacities, d.Engine.now(),
	)
	if err != nil {
		return snapshot, err
	}
	if provider, ok := d.Engine.Registry.(interface {
		Progress() []workflowv3.RegistryGenerationProgress
	}); ok {
		snapshot.RegistryGenerations = provider.Progress()
		snapshot.Queue.RegistryGenerations = snapshot.RegistryGenerations
	}
	return snapshot, nil
}

func (d *Dispatcher) QueueSnapshot(ctx context.Context) (workflowv3.QueueSnapshot, error) {
	if err := d.validate(); err != nil {
		return workflowv3.QueueSnapshot{}, err
	}
	snapshot, err := d.Engine.Store.QueueSnapshot(
		ctx, d.Engine.Registry, d.Capacities, d.Engine.now(),
	)
	if err != nil {
		return snapshot, err
	}
	if provider, ok := d.Engine.Registry.(interface {
		Progress() []workflowv3.RegistryGenerationProgress
	}); ok {
		snapshot.RegistryGenerations = provider.Progress()
	}
	return snapshot, nil
}

func (d *Dispatcher) validate() error {
	if d == nil || d.Engine == nil {
		return fmt.Errorf("workflow v3 dispatcher requires an engine")
	}
	if err := d.Engine.validate(); err != nil {
		return err
	}
	if len(d.Capacities) == 0 {
		return fmt.Errorf("workflow v3 dispatcher requires resource capacities")
	}
	for resource, capacity := range d.Capacities {
		if resource == "" || capacity < 1 {
			return fmt.Errorf("resource %q capacity must be positive", resource)
		}
	}
	return nil
}

func (d *Dispatcher) pollInterval() time.Duration {
	if d.PollInterval > 0 {
		return d.PollInterval
	}
	return 50 * time.Millisecond
}

func isRecordedAttemptError(err error) bool {
	var recorded *AttemptExecutionError
	return errors.As(err, &recorded) || errors.Is(err, workflowv3sqlite.ErrStaleCompletion)
}

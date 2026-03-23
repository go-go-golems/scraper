package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	"github.com/rs/zerolog/log"
)

type Config struct {
	MaxWorkers           int
	PollInterval         time.Duration
	DefaultLeaseDuration time.Duration
}

func (c Config) Validate() error {
	if c.MaxWorkers <= 0 {
		return fmt.Errorf("max workers must be > 0")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("poll interval must be > 0")
	}
	if c.DefaultLeaseDuration <= 0 {
		return fmt.Errorf("default lease duration must be > 0")
	}

	return nil
}

type EventKind string

const (
	EventWorkflowCreated  EventKind = "workflow_created"
	EventOpLeased         EventKind = "op_leased"
	EventOpSucceeded      EventKind = "op_succeeded"
	EventOpRetried        EventKind = "op_retried"
	EventOpFailed         EventKind = "op_failed"
	EventWorkflowUpdated  EventKind = "workflow_updated"
	EventQueueRateLimited EventKind = "queue_rate_limited"
	EventIdle             EventKind = "idle"
)

type Event struct {
	Kind       EventKind
	OccurredAt time.Time
	WorkflowID model.WorkflowID
	OpID       model.OpID
	Site       model.SiteName
	Queue      model.QueueKey
	Status     model.WorkflowStatus
	Attempt    int
	Message    string
	Error      *model.OpError
}

type Observer interface {
	OnSchedulerEvent(ctx context.Context, event Event)
}

type ObserverFunc func(ctx context.Context, event Event)

func (f ObserverFunc) OnSchedulerEvent(ctx context.Context, event Event) {
	if f != nil {
		f(ctx, event)
	}
}

type Scheduler struct {
	config              Config
	store               storecontract.Store
	runners             *runner.Registry
	workerID            string
	observer            Observer
	scraperDB           databasemod.QueryExecer
	siteDBProvider      func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error)
	queuePolicyProvider func(ctx context.Context, site model.SiteName, queue model.QueueKey) model.QueuePolicy
	now                 func() time.Time
}

func New(store storecontract.Store, runners *runner.Registry, config Config, workerID string, observer Observer) (*Scheduler, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if runners == nil {
		runners = runner.NewRegistry()
	}
	if workerID == "" {
		workerID = "scheduler-worker"
	}

	return &Scheduler{
		config:   config,
		store:    store,
		runners:  runners,
		workerID: workerID,
		observer: observer,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
}

func (s *Scheduler) SetScraperDB(db databasemod.QueryExecer) {
	if s == nil {
		return
	}
	s.scraperDB = db
}

func (s *Scheduler) SetSiteDBProvider(
	provider func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error),
) {
	if s == nil {
		return
	}
	s.siteDBProvider = provider
}

func (s *Scheduler) SetQueuePolicyProvider(
	provider func(ctx context.Context, site model.SiteName, queue model.QueueKey) model.QueuePolicy,
) {
	if s == nil {
		return
	}
	s.queuePolicyProvider = provider
}

func (s *Scheduler) SetNowFunc(now func() time.Time) {
	if s == nil || now == nil {
		return
	}
	s.now = func() time.Time {
		return now().UTC()
	}
}

func (s *Scheduler) CreateWorkflow(ctx context.Context, params storecontract.CreateWorkflowParams) error {
	workflow := params.Workflow
	if workflow.Status == "" {
		workflow.Status = model.WorkflowStatusPending
	}
	params.Workflow = workflow

	if err := s.store.CreateWorkflow(ctx, params); err != nil {
		return err
	}

	s.emit(ctx, Event{
		Kind:       EventWorkflowCreated,
		OccurredAt: s.now(),
		WorkflowID: workflow.ID,
		Site:       workflow.Site,
		Status:     workflow.Status,
		Message:    "workflow created",
	})

	return s.refreshWorkflowStatus(ctx, workflow.ID)
}

type CycleResult struct {
	Refreshed      int
	Processed      int
	Succeeded      int
	Retried        int
	Failed         int
	WorkflowEvents int
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		if _, err := s.RunOnce(ctx); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Scheduler) RunOnce(ctx context.Context) (*CycleResult, error) {
	now := s.now()
	refreshed, err := s.store.RefreshRunnableOps(ctx, now)
	if err != nil {
		return nil, err
	}

	candidates, err := s.store.ListQueueCandidates(ctx, now)
	if err != nil {
		return nil, err
	}

	result := &CycleResult{
		Refreshed: refreshed,
	}

	if len(candidates) == 0 {
		s.emit(ctx, Event{
			Kind:       EventIdle,
			OccurredAt: now,
			Message:    "no leaseable queues",
		})
		return result, nil
	}

	processedWorkflows := map[model.WorkflowID]struct{}{}
	for i, candidate := range candidates {
		if i >= s.config.MaxWorkers {
			break
		}

		policy := s.queuePolicy(ctx, candidate.Site, candidate.Queue)
		remainingGlobalSlots := s.config.MaxWorkers - result.Processed
		if remainingGlobalSlots <= 0 {
			break
		}
		maxAttempts := policy.MaxInFlight
		if maxAttempts > remainingGlobalSlots {
			maxAttempts = remainingGlobalSlots
		}

		leasedInQueue := 0
		for leasedInQueue < maxAttempts {
			op, lease, err := s.store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
				WorkerID:      s.workerID,
				Queue:         candidate.Queue,
				Site:          candidate.Site,
				Policy:        policy,
				LeaseDuration: s.config.DefaultLeaseDuration,
				Now:           now,
			})
			if err != nil {
				return nil, err
			}
			if op == nil || lease == nil {
				if leasedInQueue == 0 && policy.RateLimit != nil {
					s.emit(ctx, Event{
						Kind:       EventQueueRateLimited,
						OccurredAt: now,
						Site:       candidate.Site,
						Queue:      candidate.Queue,
						Message:    "queue had ready work but could not lease due to queue policy",
					})
				}
				break
			}

			leasedInQueue++
			result.Processed++
			processedWorkflows[op.WorkflowID] = struct{}{}
			s.emit(ctx, Event{
				Kind:       EventOpLeased,
				OccurredAt: now,
				WorkflowID: op.WorkflowID,
				OpID:       op.ID,
				Site:       op.Site,
				Queue:      op.Queue,
				Attempt:    op.RetryState.Attempt + 1,
				Message:    "leased op",
			})

			if err := s.executeLeasedOp(ctx, *op, *lease, now); err != nil {
				return nil, err
			}
		}
	}

	for workflowID := range processedWorkflows {
		if err := s.refreshWorkflowStatus(ctx, workflowID); err != nil {
			return nil, err
		}
		result.WorkflowEvents++
	}

	return result, nil
}

func (s *Scheduler) queuePolicy(ctx context.Context, site model.SiteName, queue model.QueueKey) model.QueuePolicy {
	if s.queuePolicyProvider == nil {
		return model.DefaultQueuePolicy()
	}
	return s.queuePolicyProvider(ctx, site, queue).Normalize()
}

func (s *Scheduler) executeLeasedOp(ctx context.Context, op model.OpSpec, lease model.Lease, now time.Time) error {
	workflow, err := s.store.GetWorkflow(ctx, op.WorkflowID)
	if err != nil {
		return err
	}
	if workflow == nil {
		return fmt.Errorf("workflow %s not found for op %s", op.WorkflowID, op.ID)
	}

	impl, ok := s.runners.Get(op.Kind)
	if !ok {
		opErr := model.OpError{
			Code:       "missing_runner",
			Message:    fmt.Sprintf("no runner registered for kind %q", op.Kind),
			Retryable:  false,
			OccurredAt: now,
		}
		return s.failLeasedOp(ctx, op, lease, now, opErr)
	}

	var siteDB databasemod.QueryExecer
	if s.siteDBProvider != nil {
		siteDB, err = s.siteDBProvider(ctx, op.Site)
		if err != nil {
			return fmt.Errorf("resolve site db for %s: %w", op.Site, err)
		}
	}

	opResult, runErr := impl.Run(ctx, runner.RunContext{
		Workflow:     *workflow,
		Op:           op,
		Lease:        lease,
		Now:          now,
		Dependencies: dependencyResolverAdapter{store: s.store},
		ScraperDB:    s.scraperDB,
		SiteDB:       siteDB,
	})
	if runErr != nil {
		opErr := classifyRunError(runErr, now)
		return s.failLeasedOp(ctx, op, lease, now, opErr)
	}
	if opResult == nil {
		opResult = &model.OpResult{
			OpID:        op.ID,
			CompletedAt: now,
		}
	}
	if opResult.Error != nil {
		if opResult.Error.OccurredAt.IsZero() {
			opResult.Error.OccurredAt = now
		}
		return s.failLeasedOp(ctx, op, lease, now, *opResult.Error)
	}
	if opResult.CompletedAt.IsZero() {
		opResult.CompletedAt = now
	}

	if err := s.store.CompleteOp(ctx, op.ID, storecontract.Completion{
		Lease:  lease,
		Result: *opResult,
	}); err != nil {
		return fmt.Errorf("complete op %s: %w", op.ID, err)
	}

	s.emit(ctx, Event{
		Kind:       EventOpSucceeded,
		OccurredAt: opResult.CompletedAt,
		WorkflowID: op.WorkflowID,
		OpID:       op.ID,
		Site:       op.Site,
		Queue:      op.Queue,
		Attempt:    op.RetryState.Attempt + 1,
		Message:    "op succeeded",
	})

	if _, err := s.store.RefreshRunnableOps(ctx, now); err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) failLeasedOp(ctx context.Context, op model.OpSpec, lease model.Lease, now time.Time, opErr model.OpError) error {
	retryState := nextRetryState(op, now, opErr)
	if err := s.store.FailOp(ctx, op.ID, storecontract.Failure{
		Lease:      lease,
		Error:      opErr,
		RetryState: retryState,
	}); err != nil {
		return fmt.Errorf("fail op %s: %w", op.ID, err)
	}

	eventKind := EventOpFailed
	message := "op failed"
	if retryState.NextAttemptAt != nil {
		eventKind = EventOpRetried
		message = fmt.Sprintf("op scheduled for retry at %s", retryState.NextAttemptAt.UTC().Format(time.RFC3339Nano))
	}

	s.emit(ctx, Event{
		Kind:       eventKind,
		OccurredAt: now,
		WorkflowID: op.WorkflowID,
		OpID:       op.ID,
		Site:       op.Site,
		Queue:      op.Queue,
		Attempt:    retryState.Attempt,
		Message:    message,
		Error:      &opErr,
	})

	if _, err := s.store.RefreshRunnableOps(ctx, now); err != nil {
		return err
	}

	return nil
}

func (s *Scheduler) refreshWorkflowStatus(ctx context.Context, workflowID model.WorkflowID) error {
	workflow, err := s.store.GetWorkflow(ctx, workflowID)
	if err != nil {
		return err
	}
	if workflow == nil {
		return nil
	}

	stats, err := s.store.GetWorkflowStats(ctx, workflowID)
	if err != nil {
		return err
	}
	if stats == nil || stats.Total == 0 {
		return nil
	}

	status := workflow.Status
	switch {
	case stats.Pending == 0 && stats.Ready == 0 && stats.Running == 0 && stats.Failed == 0 && stats.Canceled == 0 && stats.Succeeded == stats.Total:
		status = model.WorkflowStatusSucceeded
	case stats.Pending == 0 && stats.Ready == 0 && stats.Running == 0:
		status = model.WorkflowStatusFailed
	case stats.Succeeded > 0 || stats.Ready > 0 || stats.Running > 0:
		status = model.WorkflowStatusRunning
	default:
		status = model.WorkflowStatusPending
	}

	if status == workflow.Status {
		return nil
	}

	if err := s.store.UpdateWorkflowStatus(ctx, workflowID, status); err != nil {
		return err
	}

	s.emit(ctx, Event{
		Kind:       EventWorkflowUpdated,
		OccurredAt: s.now(),
		WorkflowID: workflowID,
		Site:       workflow.Site,
		Status:     status,
		Message:    "workflow status updated",
	})

	return nil
}

func nextRetryState(op model.OpSpec, now time.Time, opErr model.OpError) model.RetryState {
	state := op.RetryState
	state.Attempt++
	state.LastError = opErr.Message
	state.NextAttemptAt = nil

	if !opErr.Retryable {
		return state
	}
	if op.Retry.MaxAttempts <= 0 {
		return state
	}
	if state.Attempt >= op.Retry.MaxAttempts {
		return state
	}

	delay := backoffDelay(op.Retry, state.Attempt)
	nextAttempt := now.UTC().Add(delay)
	state.NextAttemptAt = &nextAttempt
	return state
}

func backoffDelay(policy model.RetryPolicy, attempt int) time.Duration {
	delay := policy.InitialBackoff
	if delay <= 0 {
		delay = time.Second
	}
	if attempt <= 1 {
		return clampDuration(delay, policy.MaxBackoff)
	}

	switch policy.BackoffKind {
	case model.BackoffKindLinear:
		delay = time.Duration(float64(delay) * float64(attempt))
	case model.BackoffKindExponential:
		multiplier := policy.Multiplier
		if multiplier <= 0 {
			multiplier = 2
		}
		result := float64(delay)
		for i := 1; i < attempt; i++ {
			result *= multiplier
		}
		delay = time.Duration(result)
	default:
	}

	return clampDuration(delay, policy.MaxBackoff)
}

func clampDuration(delay, max time.Duration) time.Duration {
	if max > 0 && delay > max {
		return max
	}
	return delay
}

type opErrorCarrier interface {
	OpError() model.OpError
}

func classifyRunError(err error, now time.Time) model.OpError {
	var carrier opErrorCarrier
	if errors.As(err, &carrier) {
		opErr := carrier.OpError()
		if opErr.OccurredAt.IsZero() {
			opErr.OccurredAt = now
		}
		return opErr
	}

	return model.OpError{
		Code:       "runner_error",
		Message:    err.Error(),
		Retryable:  false,
		OccurredAt: now,
	}
}

func (s *Scheduler) emit(ctx context.Context, event Event) {
	switch event.Kind {
	case EventOpFailed:
		logger := log.Warn()
		if event.Error != nil {
			logger = logger.Str("error_code", event.Error.Code).Bool("retryable", event.Error.Retryable)
		}
		logger.Str("event", string(event.Kind)).
			Str("workflow_id", string(event.WorkflowID)).
			Str("op_id", string(event.OpID)).
			Str("site", string(event.Site)).
			Str("queue", string(event.Queue)).
			Int("attempt", event.Attempt).
			Msg(event.Message)
	default:
		log.Info().
			Str("event", string(event.Kind)).
			Str("workflow_id", string(event.WorkflowID)).
			Str("op_id", string(event.OpID)).
			Str("site", string(event.Site)).
			Str("queue", string(event.Queue)).
			Str("workflow_status", string(event.Status)).
			Int("attempt", event.Attempt).
			Msg(event.Message)
	}

	if s.observer != nil {
		s.observer.OnSchedulerEvent(ctx, event)
	}
}

type dependencyResolverAdapter struct {
	store storecontract.Store
}

func (a dependencyResolverAdapter) Result(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) (*model.OpResult, error) {
	return a.store.GetResult(ctx, workflowID, opID)
}

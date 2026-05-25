package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/go-go-golems/scraper/pkg/services/engineview"
)

// StoreConfig opens the durable runtime store used by Runtime.
type StoreConfig interface {
	Open(context.Context) (storecontract.Store, func() error, error)
	OperatorService() OperatorService
}

type sqliteStoreConfig struct {
	path string
}

// SQLiteStore configures Runtime to use the existing SQLite engine store.
func SQLiteStore(path string) StoreConfig {
	return sqliteStoreConfig{path: path}
}

func (c sqliteStoreConfig) OperatorService() OperatorService {
	if c.path == "" {
		return nil
	}
	return engineview.NewService(c.path)
}

func (c sqliteStoreConfig) Open(ctx context.Context) (storecontract.Store, func() error, error) {
	if c.path == "" {
		return nil, nil, fmt.Errorf("sqlite workflow store path is required")
	}
	dir := filepath.Dir(c.path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, nil, fmt.Errorf("create sqlite workflow store directory: %w", err)
		}
	}
	store, err := sqlitestore.Open(ctx, c.path)
	if err != nil {
		return nil, nil, err
	}
	return store, store.Close, nil
}

// Config configures an embeddable workflow Runtime.
type Config struct {
	Store           StoreConfig
	ArtifactStore   ArtifactStore
	ProjectionStore ProjectionStore

	WorkerID      string
	MaxWorkers    int
	PollInterval  time.Duration
	LeaseDuration time.Duration

	Queues map[model.QueueKey]QueueConfig
}

type QueueConfig struct {
	MaxWorkers int
	RateLimit  *model.RateLimitPolicy
}

// Runtime is the embeddable workflow engine facade. It wraps the existing store,
// runner registry, and scheduler behind workflow-native concepts.
type Runtime struct {
	store       storecontract.Store
	closeStore  func() error
	runners     *runner.Registry
	scheduler   *scheduler.Scheduler
	operators   OperatorService
	artifacts   ArtifactStore
	projections ProjectionStore
	packages    map[string]*Package
	queues      map[model.QueueKey]QueueConfig
}

func NewRuntime(ctx context.Context, cfg Config) (*Runtime, error) {
	cfg = normalizeConfig(cfg)
	if cfg.Store == nil {
		return nil, fmt.Errorf("workflow runtime store is required")
	}
	store, closeStore, err := cfg.Store.Open(ctx)
	if err != nil {
		return nil, err
	}
	runners := runner.NewRegistry()
	s, err := scheduler.New(store, runners, scheduler.Config{
		MaxWorkers:           cfg.MaxWorkers,
		PollInterval:         cfg.PollInterval,
		DefaultLeaseDuration: cfg.LeaseDuration,
	}, cfg.WorkerID, nil)
	if err != nil {
		_ = closeStore()
		return nil, err
	}
	rt := &Runtime{
		store:       store,
		closeStore:  closeStore,
		runners:     runners,
		scheduler:   s,
		operators:   cfg.Store.OperatorService(),
		artifacts:   cfg.ArtifactStore,
		projections: cfg.ProjectionStore,
		packages:    map[string]*Package{},
		queues:      cfg.Queues,
	}
	s.SetQueuePolicyProvider(rt.queuePolicy)
	return rt, nil
}

func normalizeConfig(cfg Config) Config {
	if cfg.WorkerID == "" {
		cfg.WorkerID = "workflow-runtime"
	}
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 1
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 250 * time.Millisecond
	}
	if cfg.LeaseDuration <= 0 {
		cfg.LeaseDuration = 30 * time.Second
	}
	if cfg.Queues == nil {
		cfg.Queues = map[model.QueueKey]QueueConfig{}
	}
	return cfg
}

func (rt *Runtime) Close() error {
	if rt == nil {
		return nil
	}
	var firstErr error
	if closer, ok := rt.projections.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if rt.closeStore != nil {
		if err := rt.closeStore(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (rt *Runtime) RegisterExecutor(executor Executor) error {
	if rt == nil {
		return fmt.Errorf("workflow runtime is nil")
	}
	if executor == nil {
		return fmt.Errorf("workflow executor is nil")
	}
	if rt.artifacts != nil || rt.projections != nil {
		return rt.runners.Register(toRunnerWithStores(executor, rt.artifacts, rt.projections))
	}
	return rt.runners.Register(ToRunner(executor))
}

func (rt *Runtime) RegisterPackage(pkg *Package) error {
	if rt == nil {
		return fmt.Errorf("workflow runtime is nil")
	}
	if pkg == nil {
		return fmt.Errorf("workflow package is nil")
	}
	if pkg.Name == "" {
		return fmt.Errorf("workflow package name is required")
	}
	if pkg.Entrypoint == nil {
		return fmt.Errorf("workflow package %q entrypoint is required", pkg.Name)
	}
	rt.packages[pkg.Name] = pkg
	return nil
}

type RunOption func(*runOptions)

type runOptions struct {
	ID       string
	Name     string
	Metadata map[string]string
}

func WithRunID(id string) RunOption {
	return func(o *runOptions) { o.ID = id }
}

func WithRunName(name string) RunOption {
	return func(o *runOptions) { o.Name = name }
}

func WithRunMetadata(metadata map[string]string) RunOption {
	return func(o *runOptions) { o.Metadata = cloneStringMap(metadata) }
}

type RunHandle struct {
	ID      model.WorkflowID
	Package string
	Name    string
}

func (rt *Runtime) StartRun(ctx context.Context, packageName string, input any, opts ...RunOption) (*RunHandle, error) {
	if rt == nil {
		return nil, fmt.Errorf("workflow runtime is nil")
	}
	pkg := rt.packages[packageName]
	if pkg == nil {
		return nil, fmt.Errorf("workflow package %q is not registered", packageName)
	}
	options := runOptions{Metadata: map[string]string{}}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.ID == "" {
		options.ID = packageName + "-" + uuid.NewString()
	}
	if options.Name == "" {
		if pkg.DisplayName != "" {
			options.Name = pkg.DisplayName
		} else {
			options.Name = packageName
		}
	}
	rawInput, err := marshalJSON(input)
	if err != nil {
		return nil, fmt.Errorf("marshal run input: %w", err)
	}
	workflow := model.WorkflowRun{
		ID:       model.WorkflowID(options.ID),
		Site:     model.SiteName(packageName),
		Name:     options.Name,
		Status:   model.WorkflowStatusPending,
		Input:    rawInput,
		Metadata: cloneStringMap(options.Metadata),
	}
	builder := &RunBuilder{workflow: workflow}
	if err := pkg.Entrypoint.Start(ctx, builder, rawInput); err != nil {
		return nil, err
	}
	workflow = builder.workflow
	if err := rt.scheduler.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: workflow,
		Initial:  builder.steps,
	}); err != nil {
		return nil, err
	}
	return &RunHandle{ID: workflow.ID, Package: packageName, Name: workflow.Name}, nil
}

func (rt *Runtime) RunOnce(ctx context.Context) (*scheduler.CycleResult, error) {
	if rt == nil || rt.scheduler == nil {
		return nil, fmt.Errorf("workflow runtime scheduler is not configured")
	}
	return rt.scheduler.RunOnce(ctx)
}

// StartWorkers runs scheduler cycles until ctx is canceled. It is intentionally
// context-driven so embedded applications can use their own lifecycle manager.
func (rt *Runtime) StartWorkers(ctx context.Context, opts ...WorkerOption) error {
	if rt == nil || rt.scheduler == nil {
		return fmt.Errorf("workflow runtime scheduler is not configured")
	}
	options := workerOptions{PollInterval: 250 * time.Millisecond}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	if options.PollInterval <= 0 {
		options.PollInterval = 250 * time.Millisecond
	}
	for {
		if options.MaxCycles > 0 && options.cycles >= options.MaxCycles {
			return nil
		}
		if _, err := rt.RunOnce(ctx); err != nil {
			return err
		}
		options.cycles++
		if options.MaxCycles > 0 && options.cycles >= options.MaxCycles {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(options.PollInterval):
		}
	}
}

func (rt *Runtime) Projection(ctx context.Context, name string) (Projection, error) {
	if rt == nil || rt.projections == nil {
		return nil, fmt.Errorf("workflow runtime projection store is not configured")
	}
	return rt.projections.Projection(ctx, name)
}

func (rt *Runtime) Result(ctx context.Context, runID model.WorkflowID, stepID model.OpID) (*model.OpResult, error) {
	if rt == nil || rt.store == nil {
		return nil, fmt.Errorf("workflow runtime store is not configured")
	}
	return rt.store.GetResult(ctx, runID, stepID)
}

func (rt *Runtime) Workflow(ctx context.Context, runID model.WorkflowID) (*model.WorkflowRun, error) {
	if rt == nil || rt.store == nil {
		return nil, fmt.Errorf("workflow runtime store is not configured")
	}
	return rt.store.GetWorkflow(ctx, runID)
}

func (rt *Runtime) queuePolicy(_ context.Context, _ model.SiteName, queue model.QueueKey) model.QueuePolicy {
	cfg, ok := rt.queues[queue]
	if !ok {
		return model.DefaultQueuePolicy()
	}
	return model.QueuePolicy{MaxInFlight: cfg.MaxWorkers, RateLimit: cfg.RateLimit}.Normalize()
}

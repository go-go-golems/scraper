package workflow

import (
	"context"
	"fmt"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
)

// Executor executes one durable workflow step kind. It is the workflow-native
// facade over the lower-level engine/runner.Runner interface.
type Executor interface {
	Kind() string
	Execute(ctx context.Context, step *StepContext) error
}

// ExecutorFunc adapts a function to an Executor.
type ExecutorFunc struct {
	KindName string
	Func     func(context.Context, *StepContext) error
}

func (e ExecutorFunc) Kind() string { return e.KindName }

func (e ExecutorFunc) Execute(ctx context.Context, step *StepContext) error {
	if e.Func == nil {
		return fmt.Errorf("workflow executor %q has no function", e.KindName)
	}
	return e.Func(ctx, step)
}

// TypedExecutor executes a step after StepContext input has been decoded into I.
type TypedExecutor[I any] interface {
	Kind() string
	ExecuteTyped(ctx context.Context, step *StepContext, input I) error
}

// TypedExecutorFunc adapts a typed function to an Executor.
type TypedExecutorFunc[I any] struct {
	KindName string
	Func     func(context.Context, *StepContext, I) error
}

func (e TypedExecutorFunc[I]) Kind() string { return e.KindName }

func (e TypedExecutorFunc[I]) Execute(ctx context.Context, step *StepContext) error {
	if e.Func == nil {
		return fmt.Errorf("workflow typed executor %q has no function", e.KindName)
	}
	var input I
	if err := step.Input(&input); err != nil {
		return err
	}
	return e.Func(ctx, step, input)
}

func (e TypedExecutorFunc[I]) ExecuteTyped(ctx context.Context, step *StepContext, input I) error {
	if e.Func == nil {
		return fmt.Errorf("workflow typed executor %q has no function", e.KindName)
	}
	return e.Func(ctx, step, input)
}

// NewExecutor creates an untyped executor from a function.
func NewExecutor(kind string, fn func(context.Context, *StepContext) error) Executor {
	return ExecutorFunc{KindName: kind, Func: fn}
}

// NewTypedExecutor creates an executor that decodes step input into I before
// invoking fn.
func NewTypedExecutor[I any](kind string, fn func(context.Context, *StepContext, I) error) Executor {
	return TypedExecutorFunc[I]{KindName: kind, Func: fn}
}

// ToRunner adapts a workflow Executor to the existing engine runner interface.
func ToRunner(executor Executor) runner.Runner {
	return executorRunner{executor: executor}
}

type executorRunner struct {
	executor Executor
}

func (r executorRunner) Kind() string {
	if r.executor == nil {
		return ""
	}
	return r.executor.Kind()
}

func (r executorRunner) Run(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
	if r.executor == nil {
		return nil, fmt.Errorf("workflow executor is nil")
	}
	step := newStepContext(ctx, runCtx, nil, nil)
	if err := r.executor.Execute(ctx, step); err != nil {
		return nil, err
	}
	return step.opResult(), nil
}

func toRunnerWithStores(executor Executor, artifacts ArtifactStore, projections ProjectionStore) runner.Runner {
	return executorRunnerWithStores{executor: executor, artifacts: artifacts, projections: projections}
}

type executorRunnerWithStores struct {
	executor    Executor
	artifacts   ArtifactStore
	projections ProjectionStore
}

func (r executorRunnerWithStores) Kind() string {
	if r.executor == nil {
		return ""
	}
	return r.executor.Kind()
}

func (r executorRunnerWithStores) Run(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
	if r.executor == nil {
		return nil, fmt.Errorf("workflow executor is nil")
	}
	step := newStepContext(ctx, runCtx, r.artifacts, r.projections)
	if err := r.executor.Execute(ctx, step); err != nil {
		return nil, err
	}
	return step.opResult(), nil
}

// Registry is a small workflow-native executor registry that can produce the
// existing runner registry used by the scheduler.
type Registry struct {
	runners *runner.Registry
}

func NewRegistry() *Registry {
	return &Registry{runners: runner.NewRegistry()}
}

func (r *Registry) Register(executor Executor) error {
	if r == nil {
		return fmt.Errorf("workflow executor registry is nil")
	}
	if executor == nil {
		return fmt.Errorf("workflow executor is nil")
	}
	if executor.Kind() == "" {
		return fmt.Errorf("workflow executor kind is required")
	}
	return r.runners.Register(ToRunner(executor))
}

func (r *Registry) RunnerRegistry() *runner.Registry {
	if r == nil || r.runners == nil {
		return runner.NewRegistry()
	}
	return r.runners
}

func (r *Registry) Kinds() []string {
	if r == nil || r.runners == nil {
		return nil
	}
	return r.runners.Kinds()
}

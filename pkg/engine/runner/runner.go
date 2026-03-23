package runner

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

type DependencyResolver interface {
	Result(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) (*model.OpResult, error)
}

type SiteDatabase interface {
	Exec(ctx context.Context, sql string, args ...any) error
}

type RunContext struct {
	Workflow     model.WorkflowRun
	Op           model.OpSpec
	Lease        model.Lease
	Now          time.Time
	Dependencies DependencyResolver
	SiteDB       SiteDatabase
}

type Runner interface {
	Kind() string
	Run(ctx context.Context, runCtx RunContext) (*model.OpResult, error)
}

type Registry struct {
	runners map[string]Runner
}

func NewRegistry() *Registry {
	return &Registry{
		runners: map[string]Runner{},
	}
}

func (r *Registry) Register(runner Runner) error {
	kind := runner.Kind()
	if _, ok := r.runners[kind]; ok {
		return fmt.Errorf("runner already registered for kind %q", kind)
	}
	r.runners[kind] = runner
	return nil
}

func (r *Registry) Get(kind string) (Runner, bool) {
	runner, ok := r.runners[kind]
	return runner, ok
}

func (r *Registry) Kinds() []string {
	ret := make([]string, 0, len(r.runners))
	for kind := range r.runners {
		ret = append(ret, kind)
	}
	sort.Strings(ret)
	return ret
}

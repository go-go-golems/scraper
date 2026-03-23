package runner

import (
	"context"
	"fmt"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	scraperjsruntime "github.com/go-go-golems/scraper/pkg/js/runtime"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
)

type JSRunner struct {
	registry *siteregistry.Registry
}

func NewJSRunner(registry *siteregistry.Registry) *JSRunner {
	if registry == nil {
		registry = siteregistry.New()
	}

	return &JSRunner{
		registry: registry,
	}
}

func (r *JSRunner) Kind() string {
	return "js"
}

func (r *JSRunner) Run(ctx context.Context, runCtx RunContext) (*model.OpResult, error) {
	def, ok := r.registry.Get(runCtx.Op.Site)
	if !ok {
		return nil, fmt.Errorf("site %q is not registered", runCtx.Op.Site)
	}

	executor := scraperjsruntime.NewExecutor(scraperjsruntime.ExecutorConfig{
		ScriptsFS:               def.ScriptsFS,
		ScriptsRoot:             def.ScriptsRoot,
		Modules:                 def.Modules,
		RuntimeModuleRegistrars: def.RuntimeModuleRegistrars,
		ScraperDB:               runCtx.ScraperDB,
		SiteDB:                  runCtx.SiteDB,
	})

	return executor.Execute(ctx, scraperjsruntime.ExecutionRequest{
		Workflow:     runCtx.Workflow,
		Op:           runCtx.Op,
		Lease:        runCtx.Lease,
		Now:          runCtx.Now,
		Dependencies: runCtx.Dependencies,
	})
}

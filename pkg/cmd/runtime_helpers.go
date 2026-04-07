package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	"github.com/go-go-golems/scraper/pkg/engine/config"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/go-go-golems/scraper/pkg/metrics"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	sitemigrate "github.com/go-go-golems/scraper/pkg/sites/migrate"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
)

func ensureEngineDBDir(engineDB string) error {
	dir := filepath.Dir(engineDB)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create engine db directory: %w", err)
	}
	return nil
}

func openEngineStore(ctx context.Context, engineDB string) (*sqlitestore.Store, error) {
	if err := ensureEngineDBDir(engineDB); err != nil {
		return nil, err
	}
	return sqlitestore.Open(ctx, engineDB)
}

func openScraperDB(engineDB string) (*sql.DB, error) {
	if err := ensureEngineDBDir(engineDB); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", engineDB)
	if err != nil {
		return nil, fmt.Errorf("open scraper db: %w", err)
	}
	return db, nil
}

func newDefaultRunnerRegistry(
	siteRegistry *siteregistry.Registry,
	httpConfig config.HTTP,
	eventPublisher *runtimeevents.Publisher,
	metricsRegistry *metrics.Registry,
	component string,
	workerID string,
) (*runner.Registry, error) {
	runners := runner.NewRegistry()
	httpRunner, err := runner.NewHTTPRunner(httpConfig, nil)
	if err != nil {
		return nil, err
	}
	if err := runners.Register(runtimeevents.WrapRunner(metrics.WrapRunner(httpRunner, metricsRegistry), eventPublisher, component, workerID)); err != nil {
		return nil, err
	}
	if err := runners.Register(runtimeevents.WrapRunner(metrics.WrapRunner(runner.NewJSRunner(siteRegistry), metricsRegistry), eventPublisher, component, workerID)); err != nil {
		return nil, err
	}
	return runners, nil
}

type siteDBProvider struct {
	manager  *sitemigrate.Manager
	sitesDir string
	dbs      map[model.SiteName]*sql.DB
}

func newSiteDBProvider(siteRegistry *siteregistry.Registry, sitesDir string) *siteDBProvider {
	return &siteDBProvider{
		manager:  sitemigrate.NewManager(siteRegistry),
		sitesDir: sitesDir,
		dbs:      map[model.SiteName]*sql.DB{},
	}
}

func (p *siteDBProvider) QueryExecer(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error) {
	if db, ok := p.dbs[site]; ok {
		return db, nil
	}

	report, err := p.manager.Migrate(ctx, site, p.sitesDir)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", report.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("open site db: %w", err)
	}
	p.dbs[site] = db
	return db, nil
}

func (p *siteDBProvider) Close() error {
	var firstErr error
	for site, db := range p.dbs {
		if err := db.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close site db %s: %w", site, err)
		}
		delete(p.dbs, site)
	}
	return firstErr
}

type workerLoopResult struct {
	Cycles    int
	Refreshed int
	Processed int
	Succeeded int
	Retried   int
	Failed    int
}

type schedulerRunner interface {
	RunOnce(ctx context.Context) (*scheduler.CycleResult, error)
}

func runSchedulerCycles(ctx context.Context, s schedulerRunner, pollInterval time.Duration, maxCycles int, metricsRegistry *metrics.Registry, workerID string) (*workerLoopResult, error) {
	ret := &workerLoopResult{}
	for {
		if maxCycles > 0 && ret.Cycles >= maxCycles {
			return ret, nil
		}

		cycleStarted := time.Now()
		result, err := s.RunOnce(ctx)
		if err != nil {
			return nil, err
		}
		metricsRegistry.ObserveSchedulerCycle(workerID, time.Since(cycleStarted))

		ret.Cycles++
		ret.Refreshed += result.Refreshed
		ret.Processed += result.Processed
		ret.Succeeded += result.Succeeded
		ret.Retried += result.Retried
		ret.Failed += result.Failed

		if maxCycles > 0 && ret.Cycles >= maxCycles {
			return ret, nil
		}

		select {
		case <-ctx.Done():
			return ret, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

func setSchedulerSiteRuntime(
	s *scheduler.Scheduler,
	siteRegistry *siteregistry.Registry,
	scraperDB databasemod.QueryExecer,
	siteDBProvider func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error),
) {
	if s == nil {
		return
	}
	if scraperDB != nil {
		s.SetScraperDB(scraperDB)
	}
	if siteDBProvider != nil {
		s.SetSiteDBProvider(siteDBProvider)
	}
	if siteRegistry != nil {
		s.SetQueuePolicyProvider(siteRegistry.QueuePolicyProvider())
	}
}

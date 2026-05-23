package cmd

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/config"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	"github.com/go-go-golems/scraper/pkg/metrics"
	runtimestream "github.com/go-go-golems/scraper/pkg/runtimeevents/sessionstream"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
)

func runWorkerCommand(ctx context.Context, out io.Writer, options *workerCommandOptions, siteRegistry *siteregistry.Registry) error {
	cfg := config.Config{
		Paths: config.Paths{
			EngineDB: options.engineDB,
			SitesDir: options.sitesDir,
		},
		Worker: config.Worker{
			WorkerID:             options.workerID,
			MaxConcurrentOps:     options.maxWorkers,
			PollInterval:         options.pollInterval,
			DefaultLeaseDuration: options.leaseDuration,
		},
		HTTP: config.HTTP{
			UserAgent: "scraper/worker",
			Timeout:   options.httpTimeout,
			ProxyURL:  options.httpProxy,
		},
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	store, err := openEngineStore(ctx, options.engineDB)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	scraperDB, err := openScraperDB(options.engineDB)
	if err != nil {
		return err
	}
	defer func() { _ = scraperDB.Close() }()

	eventConfig, err := options.runtimeEvents.publisherConfig()
	if err != nil {
		return err
	}
	eventRuntime, err := runtimestream.NewProducerRuntime(runtimestream.Config{Events: eventConfig})
	if err != nil {
		return err
	}
	defer func() { _ = eventRuntime.Close(context.Background()) }()

	eventPublisher := eventRuntime.Publisher
	metricsRegistry, err := metrics.NewRegistry()
	if err != nil {
		return err
	}
	metricsRegistry.SetWorkerUp(options.workerID, true)
	defer metricsRegistry.SetWorkerUp(options.workerID, false)

	runners, err := newDefaultRunnerRegistry(siteRegistry, cfg.HTTP, eventPublisher, metricsRegistry, "worker-runner", options.workerID)
	if err != nil {
		return err
	}

	siteDBs := newSiteDBProvider(siteRegistry, options.sitesDir)
	defer func() { _ = siteDBs.Close() }()

	observer := newWorkerObserver(eventPublisher, metricsRegistry, options.workerID)

	s, err := scheduler.New(store, runners, scheduler.Config{
		MaxWorkers:           options.maxWorkers,
		PollInterval:         options.pollInterval,
		DefaultLeaseDuration: options.leaseDuration,
	}, options.workerID, observer)
	if err != nil {
		return err
	}
	setSchedulerSiteRuntime(s, siteRegistry, scraperDB, siteDBs.QueryExecer)

	metricsServer, _, err := maybeStartWorkerMetricsServer(ctx, options.metricsAddr, options.metricsPath, metricsRegistry)
	if err != nil {
		return err
	}
	if metricsServer != nil {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_ = metricsServer.Shutdown(shutdownCtx)
		}()
	}

	result, err := runSchedulerCycles(ctx, s, options.pollInterval, options.maxCycles, metricsRegistry, options.workerID)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "Worker ID: %s\n", options.workerID)
	_, _ = fmt.Fprintf(out, "Engine DB: %s\n", options.engineDB)
	_, _ = fmt.Fprintf(out, "Sites dir: %s\n", options.sitesDir)
	_, _ = fmt.Fprintf(out, "Cycles: %d\n", result.Cycles)
	_, _ = fmt.Fprintf(out, "Refreshed: %d\n", result.Refreshed)
	_, _ = fmt.Fprintf(out, "Processed: %d\n", result.Processed)
	_, _ = fmt.Fprintf(out, "Succeeded: %d\n", result.Succeeded)
	_, _ = fmt.Fprintf(out, "Retried: %d\n", result.Retried)
	_, _ = fmt.Fprintf(out, "Failed: %d\n", result.Failed)
	return nil
}

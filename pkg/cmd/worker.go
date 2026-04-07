package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/config"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	"github.com/go-go-golems/scraper/pkg/metrics"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

type workerCommandOptions struct {
	engineDB      string
	sitesDir      string
	workerID      string
	maxWorkers    int
	pollInterval  time.Duration
	leaseDuration time.Duration
	httpTimeout   time.Duration
	httpProxy     string
	maxCycles     int
	metricsAddr   string
	metricsPath   string
	runtimeEvents runtimeEventOptions
}

func newWorkerCommand(siteRegistry *siteregistry.Registry) *cobra.Command {
	options := &workerCommandOptions{}

	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Run durable background workers that poll the engine DB for ready ops",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Poll the engine DB and execute ready ops",
		RunE: func(cmd *cobra.Command, args []string) error {
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

			ctx := cmd.Context()
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
			eventResources, err := runtimeevents.OpenPublisher(eventConfig)
			if err != nil {
				return err
			}
			defer func() { _ = eventResources.Close() }()

			eventPublisher := eventResources.EventPublisher()
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

			observer := composeSchedulerObservers(
				runtimeevents.NewSchedulerObserver(eventPublisher, "worker-scheduler", options.workerID),
				metrics.NewSchedulerObserver(metricsRegistry),
			)

			s, err := scheduler.New(store, runners, scheduler.Config{
				MaxWorkers:           options.maxWorkers,
				PollInterval:         options.pollInterval,
				DefaultLeaseDuration: options.leaseDuration,
			}, options.workerID, observer)
			if err != nil {
				return err
			}
			setSchedulerSiteRuntime(s, siteRegistry, scraperDB, siteDBs.QueryExecer)

			metricsServer, err := maybeStartWorkerMetricsServer(ctx, options.metricsAddr, options.metricsPath, metricsRegistry)
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Worker ID: %s\n", options.workerID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Engine DB: %s\n", options.engineDB)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Sites dir: %s\n", options.sitesDir)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cycles: %d\n", result.Cycles)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Refreshed: %d\n", result.Refreshed)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Processed: %d\n", result.Processed)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Succeeded: %d\n", result.Succeeded)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Retried: %d\n", result.Retried)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Failed: %d\n", result.Failed)
			return nil
		},
	}

	runCmd.Flags().StringVar(&options.engineDB, "engine-db", "state/engine.db", "Path to the durable engine SQLite database")
	runCmd.Flags().StringVar(&options.sitesDir, "sites-dir", "state/sites", "Directory that stores per-site SQLite databases")
	runCmd.Flags().StringVar(&options.workerID, "worker-id", "scraper-worker", "Stable worker identifier written into op leases")
	runCmd.Flags().IntVar(&options.maxWorkers, "max-workers", 4, "Maximum number of queue domains processed per scheduler cycle")
	runCmd.Flags().DurationVar(&options.pollInterval, "poll-interval", 250*time.Millisecond, "Delay between scheduler cycles")
	runCmd.Flags().DurationVar(&options.leaseDuration, "lease-duration", 30*time.Second, "Default lease duration for newly claimed ops")
	runCmd.Flags().DurationVar(&options.httpTimeout, "http-timeout", 15*time.Second, "HTTP timeout used by worker-side http/fetch ops")
	runCmd.Flags().StringVar(&options.httpProxy, "http-proxy", "", "Explicit HTTP proxy URL used by worker-side http/fetch ops")
	runCmd.Flags().IntVar(&options.maxCycles, "max-cycles", 0, "Maximum scheduler cycles to execute before exiting (0 means keep polling)")
	runCmd.Flags().StringVar(&options.metricsAddr, "metrics-address", "", "Optional address for exposing worker Prometheus metrics, for example 127.0.0.1:9091")
	runCmd.Flags().StringVar(&options.metricsPath, "metrics-path", "/metrics", "HTTP path used by the worker Prometheus metrics listener")
	addRuntimeEventFlags(runCmd, &options.runtimeEvents, false, false)

	cmd.AddCommand(runCmd)
	return cmd
}

func composeSchedulerObservers(observers ...scheduler.Observer) scheduler.Observer {
	filtered := make([]scheduler.Observer, 0, len(observers))
	for _, observer := range observers {
		if observer != nil {
			filtered = append(filtered, observer)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return scheduler.ObserverFunc(func(ctx context.Context, event scheduler.Event) {
		for _, observer := range filtered {
			observer.OnSchedulerEvent(ctx, event)
		}
	})
}

func maybeStartWorkerMetricsServer(ctx context.Context, addr string, path string, metricsRegistry *metrics.Registry) (*http.Server, error) {
	if addr == "" || metricsRegistry == nil {
		return nil, nil
	}
	if path == "" {
		path = "/metrics"
	}
	mux := http.NewServeMux()
	mux.Handle(path, metricsRegistry.Handler())
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Warn().Err(err).Str("component", "worker-metrics").Str("address", addr).Msg("worker metrics server stopped")
		}
	}()
	return server, nil
}

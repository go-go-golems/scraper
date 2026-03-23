package cmd

import (
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/config"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
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
	maxCycles     int
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

			runners, err := newDefaultRunnerRegistry(siteRegistry, cfg.HTTP)
			if err != nil {
				return err
			}

			siteDBs := newSiteDBProvider(siteRegistry, options.sitesDir)
			defer func() { _ = siteDBs.Close() }()

			s, err := scheduler.New(store, runners, scheduler.Config{
				MaxWorkers:           options.maxWorkers,
				PollInterval:         options.pollInterval,
				DefaultLeaseDuration: options.leaseDuration,
			}, options.workerID, nil)
			if err != nil {
				return err
			}
			setSchedulerSiteRuntime(s, siteRegistry, scraperDB, siteDBs.QueryExecer)

			result, err := runSchedulerCycles(ctx, s, options.pollInterval, options.maxCycles)
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
	runCmd.Flags().IntVar(&options.maxCycles, "max-cycles", 0, "Maximum scheduler cycles to execute before exiting (0 means keep polling)")

	cmd.AddCommand(runCmd)
	return cmd
}

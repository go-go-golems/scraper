package cmd

import (
	"time"

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
			// Load external sites from --sites-manifest-dir.
			if err := LoadSitesFromFlag(cmd, siteRegistry); err != nil {
				return err
			}
			return runWorkerCommand(cmd.Context(), cmd.OutOrStdout(), options, siteRegistry)
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

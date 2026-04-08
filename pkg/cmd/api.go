package cmd

import (
	"errors"
	"net/http"
	"time"

	apiserver "github.com/go-go-golems/scraper/pkg/api/server"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/spf13/cobra"
)

type apiCommandOptions struct {
	address       string
	engineDB     string
	sitesDir     string
	readTimeout  time.Duration
	writeTimeout time.Duration
	runtimeEvents runtimeEventOptions
}

func newAPICommand(version string, siteRegistry *siteregistry.Registry) *cobra.Command {
	options := &apiCommandOptions{}

	apiCmd := &cobra.Command{
		Use:   "api",
		Short: "Serve an HTTP API for durable workflow submission and inspection",
	}

	serveCmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the local HTTP API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load external sites from --sites-manifest-dir.
			if err := LoadSitesFromFlag(cmd, siteRegistry); err != nil {
				return err
			}

			eventConfig, err := options.runtimeEvents.pubSubConfig()
			if err != nil {
				return err
			}

			server, err := apiserver.New(apiserver.Config{
				Address:          options.address,
				EngineDB:         options.engineDB,
				SitesDir:         options.sitesDir,
				ReadTimeout:      options.readTimeout,
				WriteTimeout:     options.writeTimeout,
				Version:          version,
				RuntimeEvents:    eventConfig,
				RecentEventLimit: options.runtimeEvents.recentEventLimit,
			}, siteRegistry)
			if err != nil {
				return err
			}
			err = server.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		},
	}

	serveCmd.Flags().StringVar(&options.address, "address", "127.0.0.1:8080", "Address to bind the HTTP API server to")
	serveCmd.Flags().StringVar(&options.engineDB, "engine-db", "state/engine.db", "Path to the durable engine SQLite database")
	serveCmd.Flags().StringVar(&options.sitesDir, "sites-dir", "state/sites", "Directory that stores per-site SQLite databases")
	serveCmd.Flags().DurationVar(&options.readTimeout, "read-timeout", 10*time.Second, "HTTP server read timeout")
	serveCmd.Flags().DurationVar(&options.writeTimeout, "write-timeout", 30*time.Second, "HTTP server write timeout")
	addRuntimeEventFlags(serveCmd, &options.runtimeEvents, true, true)

	apiCmd.AddCommand(serveCmd)
	return apiCmd
}

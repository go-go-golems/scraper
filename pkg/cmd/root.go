package cmd

import (
	"fmt"
	"os"

	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	helpcmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	scraperdoc "github.com/go-go-golems/scraper/pkg/doc"
	"github.com/go-go-golems/scraper/pkg/sites/defaults"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/spf13/cobra"
)

const SitesManifestDirFlag = "sites-manifest-dir"

// LoadSitesFromFlag loads external sites from the --sites-manifest-dir flag into the registry.
// Returns nil if the flag is unset or empty. Shared by subcommands that need site definitions.
func LoadSitesFromFlag(cmd *cobra.Command, registry *siteregistry.Registry) error {
	dir, err := cmd.Flags().GetString(SitesManifestDirFlag)
	if err != nil {
		// Flag not found on this command — skip silently.
		return nil
	}
	if dir == "" {
		return nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("--%s %s: %w", SitesManifestDirFlag, dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("--%s %s: not a directory", SitesManifestDirFlag, dir)
	}
	return defaults.LoadExternalSites(registry, dir)
}

func NewRootCommand(version string) (*cobra.Command, error) {
	siteRegistry, err := defaults.NewRegistry()
	if err != nil {
		return nil, err
	}

	return newRootCommand(version, siteRegistry)
}

// NewRootCommandWithRegistry is like NewRootCommand but uses a pre-built registry.
// Useful for tests that need a specific set of sites loaded.
func NewRootCommandWithRegistry(version string, siteRegistry *siteregistry.Registry) (*cobra.Command, error) {
	return newRootCommand(version, siteRegistry)
}

func newRootCommand(version string, siteRegistry *siteregistry.Registry) (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:     "scraper",
		Short:   "Durable workflow-driven scraping with Go and embedded JavaScript",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.InitLoggerFromCobra(cmd)
		},
	}

	if err := logging.AddLoggingSectionToRootCommand(rootCmd, "scraper"); err != nil {
		return nil, err
	}

	rootCmd.PersistentFlags().String(SitesManifestDirFlag, "", "Directory containing external site manifests (site.yaml per subdirectory). Loaded in addition to built-in sites.")

	helpSystem := help.NewHelpSystem()
	if err := scraperdoc.AddDocToHelpSystem(helpSystem); err != nil {
		return nil, err
	}
	helpcmd.SetupCobraRootCommand(helpSystem, rootCmd)

	rootCmd.AddCommand(newEngineCommand())
	rootCmd.AddCommand(newWorkerCommand(siteRegistry))
	rootCmd.AddCommand(newAPICommand(version, siteRegistry))
	siteCmd, err := newSiteCommand(siteRegistry)
	if err != nil {
		return nil, err
	}
	rootCmd.AddCommand(siteCmd)
	rootCmd.AddCommand(newVersionCommand(version))

	return rootCmd, nil
}

package cmd

import (
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	helpcmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	scraperdoc "github.com/go-go-golems/scraper/pkg/doc"
	"github.com/go-go-golems/scraper/pkg/sites/defaults"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/spf13/cobra"
)

const SitesManifestDirFlag = "sites-manifest-dir"

// NewRootCommand creates the root scraper command with site manifests loaded from dirs.
// Pass zero or more directories containing site.yaml subdirectories.
func NewRootCommand(version string, manifestDirs ...string) (*cobra.Command, error) {
	siteRegistry, err := defaults.NewRegistryFromDirs(manifestDirs...)
	if err != nil {
		return nil, err
	}

	return newRootCommand(version, siteRegistry, manifestDirs...)
}

// NewRootCommandWithRegistry is like NewRootCommand but uses a pre-built registry.
// Useful for tests that need a specific set of sites loaded.
func NewRootCommandWithRegistry(version string, siteRegistry *siteregistry.Registry) (*cobra.Command, error) {
	return newRootCommand(version, siteRegistry)
}

func newRootCommand(version string, siteRegistry *siteregistry.Registry, manifestDirs ...string) (*cobra.Command, error) {
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

	rootCmd.PersistentFlags().StringSlice(
		SitesManifestDirFlag,
		manifestDirs,
		"Directory/directories containing site manifests (site.yaml per subdirectory). Resolved during bootstrap before the command tree is built.",
	)

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

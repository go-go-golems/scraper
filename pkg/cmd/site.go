package cmd

import (
	"fmt"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	sitemigrate "github.com/go-go-golems/scraper/pkg/sites/migrate"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/spf13/cobra"
)

type siteCommandOptions struct {
	sitesDir string
}

func newSiteCommand(siteRegistry *siteregistry.Registry) *cobra.Command {
	options := &siteCommandOptions{}
	manager := sitemigrate.NewManager(siteRegistry)

	siteCmd := &cobra.Command{
		Use:   "site",
		Short: "Manage site-specific databases and migrations",
	}
	siteCmd.PersistentFlags().StringVar(
		&options.sitesDir,
		"sites-dir",
		"state/sites",
		"Directory that stores per-site SQLite databases",
	)

	migrateCmd := &cobra.Command{
		Use:   "migrate <site>",
		Short: "Apply SQL and JS migrations for a site database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			site := model.SiteName(args[0])
			report, err := manager.Migrate(cmd.Context(), site, options.sitesDir)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Site: %s\n", report.Site)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "DB: %s\n", report.DatabasePath)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Applied migrations: %d\n", report.AppliedCount)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Current schema version: %d\n", report.CurrentVersion)
			return nil
		},
	}

	siteCmd.AddCommand(migrateCmd)

	for _, def := range siteRegistry.List() {
		if def.RegisterCLI == nil {
			continue
		}
		_ = def.RegisterCLI(siteCmd)
	}

	return siteCmd
}

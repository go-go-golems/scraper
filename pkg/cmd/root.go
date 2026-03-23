package cmd

import (
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/glazed/pkg/help"
	helpcmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	scraperdoc "github.com/go-go-golems/scraper/pkg/doc"
	"github.com/spf13/cobra"
)

func NewRootCommand(version string) (*cobra.Command, error) {
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

	helpSystem := help.NewHelpSystem()
	if err := scraperdoc.AddDocToHelpSystem(helpSystem); err != nil {
		return nil, err
	}
	helpcmd.SetupCobraRootCommand(helpSystem, rootCmd)

	rootCmd.AddCommand(newVersionCommand(version))

	return rootCmd, nil
}

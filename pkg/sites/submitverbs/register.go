package submitverbs

import (
	"context"
	"fmt"
	"io"
	"strings"

	glazedcli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/spf13/cobra"
)

type CommandOptions struct {
	EngineDB   string
	WorkflowID string
}

func RegisterSite(root *cobra.Command, siteRegistry *siteregistry.Registry, def siteregistry.Definition) error {
	if root == nil {
		return fmt.Errorf("site root command is nil")
	}
	if siteRegistry == nil {
		return fmt.Errorf("site registry is nil")
	}
	if def.VerbsFS == nil || strings.TrimSpace(def.VerbsRoot) == "" {
		return nil
	}

	registry, err := jsverbs.ScanFS(def.VerbsFS, def.VerbsRoot, jsverbs.ScanOptions{
		IncludePublicFunctions: false,
		Extensions:             []string{".js"},
		FailOnErrorDiagnostics: true,
	})
	if err != nil {
		return fmt.Errorf("scan %s submit verbs: %w", def.Name, err)
	}

	commands, err := registry.Commands()
	if err != nil {
		return fmt.Errorf("build %s submit verb commands: %w", def.Name, err)
	}

	siteCmd := findOrCreateChild(root, string(def.Name), fmt.Sprintf("Commands for %s", def.Name))
	runCmd := findOrCreateChild(siteCmd, "run", fmt.Sprintf("Submit %s workflows from JS verbs", def.Name))

	options := &CommandOptions{}
	runCmd.PersistentFlags().StringVar(
		&options.EngineDB,
		"engine-db",
		"state/engine.db",
		"Path to the durable engine SQLite database",
	)
	runCmd.PersistentFlags().StringVar(
		&options.WorkflowID,
		"workflow-id",
		"",
		"Workflow ID to use for the submitted workflow",
	)

	host := NewHost(siteRegistry, def, registry, nil)
	verbsBySource := map[string]*jsverbs.VerbSpec{}
	for _, verb := range registry.Verbs() {
		verbsBySource[verb.SourceRef()] = verb
	}
	for _, scannedCommand := range commands {
		sourceRef := strings.TrimPrefix(scannedCommand.Description().Source, "jsverbs:")
		verb, ok := verbsBySource[sourceRef]
		if !ok {
			return fmt.Errorf("verb lookup failed for %s", scannedCommand.Description().Source)
		}

		command := &commandDescriptionWrapper{
			description: scannedCommand.Description().Clone(true, cmds.WithParents()),
		}

		var cobraCmd *cobra.Command
		cobraCmd, err = glazedcli.BuildCobraCommandFromCommandAndFunc(
			command,
			func(ctx context.Context, parsedValues *values.Values) error {
				sitesDir := cobraCmd.Flag("sites-dir").Value.String()
				result, err := host.Submit(ctx, verb, parsedValues, SubmitOptions{
					EngineDB:   options.EngineDB,
					SitesDir:   sitesDir,
					WorkflowID: options.WorkflowID,
				})
				if err != nil {
					return err
				}
				result.CommandPath = fmt.Sprintf("site %s run %s", def.Name, command.Description().Name)
				return PrintSubmitResult(cobraCmd.OutOrStdout(), result, options.EngineDB)
			},
			glazedcli.WithSkipCommandSettingsSection(),
		)
		if err != nil {
			return err
		}

		runCmd.AddCommand(cobraCmd)
	}

	return nil
}

type commandDescriptionWrapper struct {
	description *cmds.CommandDescription
}

func (c *commandDescriptionWrapper) Description() *cmds.CommandDescription {
	return c.description
}

func (c *commandDescriptionWrapper) ToYAML(w io.Writer) error {
	return c.description.ToYAML(w)
}

func findOrCreateChild(parent *cobra.Command, use string, short string) *cobra.Command {
	for _, child := range parent.Commands() {
		if child.Name() == use {
			return child
		}
	}

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
	}
	parent.AddCommand(cmd)
	return cmd
}

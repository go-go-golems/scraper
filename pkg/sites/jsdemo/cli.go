package jsdemo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	sitemigrate "github.com/go-go-golems/scraper/pkg/sites/migrate"
	"github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

type cliOptions struct {
	engineDB   string
	workflowID string
	count      int
	multiplier int
	prefix     string
	index      int
	maxCycles  int
}

func registerCLI(root *cobra.Command) error {
	options := &cliOptions{}

	jsDemoCmd := &cobra.Command{
		Use:   "js-demo",
		Short: "Run the built-in pure-JavaScript demo site",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run named pure-JS js-demo entrypoints",
	}

	addSharedFlags := func(cmd *cobra.Command) {
		cmd.Flags().StringVar(
			&options.engineDB,
			"engine-db",
			"state/engine.db",
			"Path to the durable engine SQLite database",
		)
		cmd.Flags().StringVar(
			&options.workflowID,
			"workflow-id",
			"",
			"Workflow ID to use for the run (defaults to a timestamped ID)",
		)
		cmd.Flags().IntVar(
			&options.count,
			"count",
			4,
			"Number of item ops to emit",
		)
		cmd.Flags().IntVar(
			&options.multiplier,
			"multiplier",
			3,
			"Multiplier used by the generated item scripts",
		)
		cmd.Flags().StringVar(
			&options.prefix,
			"prefix",
			"demo",
			"Prefix used when generating demo item labels",
		)
		cmd.Flags().IntVar(
			&options.maxCycles,
			"max-cycles",
			32,
			"Maximum scheduler cycles before the command gives up",
		)
	}

	seedCmd := &cobra.Command{
		Use:   "seed",
		Short: "Run the full fan-out js-demo workflow starting at seed.js",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDemoWorkflow(cmd, options, "seed", BuildSeedWorkflow)
		},
	}
	addSharedFlags(seedCmd)

	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "Run the build_item.js op directly as a single-op workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDemoWorkflow(cmd, options, "item", BuildItemWorkflow)
		},
	}
	addSharedFlags(itemCmd)
	itemCmd.Flags().IntVar(
		&options.index,
		"index",
		0,
		"Zero-based item index to generate",
	)

	summaryCmd := &cobra.Command{
		Use:   "summary",
		Short: "Run the summarize.js join stage with generated item dependencies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDemoWorkflow(cmd, options, "summary", BuildSummaryWorkflow)
		},
	}
	addSharedFlags(summaryCmd)

	runCmd.AddCommand(seedCmd)
	runCmd.AddCommand(itemCmd)
	runCmd.AddCommand(summaryCmd)
	jsDemoCmd.AddCommand(runCmd)
	root.AddCommand(jsDemoCmd)
	return nil
}

type workflowBuilder func(RunOptions) (storecontract.CreateWorkflowParams, model.OpID, error)

func runDemoWorkflow(cmd *cobra.Command, options *cliOptions, entrypoint string, build workflowBuilder) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if options == nil {
		options = &cliOptions{}
	}

	registry := registry.New()
	if err := Register(registry); err != nil {
		return err
	}

	manager := sitemigrate.NewManager(registry)
	report, err := manager.Migrate(ctx, model.SiteName("js-demo"), cmd.Flag("sites-dir").Value.String())
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(options.engineDB), 0o755); err != nil {
		return fmt.Errorf("create engine db directory: %w", err)
	}

	engineStore, err := sqlitestore.Open(ctx, options.engineDB)
	if err != nil {
		return err
	}
	defer func() { _ = engineStore.Close() }()

	siteDB, err := sql.Open("sqlite3", report.DatabasePath)
	if err != nil {
		return fmt.Errorf("open site db: %w", err)
	}
	defer func() { _ = siteDB.Close() }()

	runners := runner.NewRegistry()
	if err := runners.Register(runner.NewJSRunner(registry)); err != nil {
		return err
	}

	s, err := scheduler.New(engineStore, runners, scheduler.Config{
		MaxWorkers:           8,
		PollInterval:         25 * time.Millisecond,
		DefaultLeaseDuration: 30 * time.Second,
	}, "js-demo-cli", nil)
	if err != nil {
		return err
	}
	s.SetSiteDBProvider(func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error) {
		if site != model.SiteName("js-demo") {
			return nil, fmt.Errorf("unexpected site db request for %s", site)
		}
		return siteDB, nil
	})

	workflowParams, targetOpID, err := build(RunOptions{
		WorkflowID: options.workflowID,
		Count:      options.count,
		Multiplier: options.multiplier,
		Prefix:     options.prefix,
		Index:      options.index,
	})
	if err != nil {
		return err
	}

	if err := s.CreateWorkflow(ctx, workflowParams); err != nil {
		return err
	}

	maxCycles := options.maxCycles
	if maxCycles <= 0 {
		maxCycles = 32
	}

	var workflow *model.WorkflowRun
	for i := 0; i < maxCycles; i++ {
		if _, err := s.RunOnce(ctx); err != nil {
			return err
		}

		workflow, err = engineStore.GetWorkflow(ctx, workflowParams.Workflow.ID)
		if err != nil {
			return err
		}
		if workflow != nil && workflow.Status == model.WorkflowStatusSucceeded {
			break
		}
		if workflow != nil && workflow.Status == model.WorkflowStatusFailed {
			return fmt.Errorf("workflow %s failed", workflow.ID)
		}
	}

	if workflow == nil {
		return fmt.Errorf("workflow %s was not created", workflowParams.Workflow.ID)
	}
	if workflow.Status != model.WorkflowStatusSucceeded {
		return fmt.Errorf("workflow %s did not finish after %d cycles (status=%s)", workflow.ID, maxCycles, workflow.Status)
	}

	result, err := engineStore.GetResult(ctx, workflow.ID, targetOpID)
	if err != nil {
		return err
	}
	if result == nil {
		return fmt.Errorf("%s result %s not found", entrypoint, targetOpID)
	}

	prettySummary := "{}"
	if len(result.Data) > 0 {
		var decoded any
		if err := json.Unmarshal(result.Data, &decoded); err == nil {
			if pretty, err := json.MarshalIndent(decoded, "", "  "); err == nil {
				prettySummary = string(pretty)
			}
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Site: js-demo\n")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Entrypoint: %s\n", entrypoint)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Workflow: %s\n", workflow.ID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", workflow.Status)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Engine DB: %s\n", options.engineDB)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Site DB: %s\n", report.DatabasePath)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Result op: %s\n", targetOpID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Result:\n%s\n", prettySummary)

	return nil
}

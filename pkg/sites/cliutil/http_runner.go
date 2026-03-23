package cliutil

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	"github.com/go-go-golems/scraper/pkg/engine/config"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	sitemigrate "github.com/go-go-golems/scraper/pkg/sites/migrate"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

type HTTPWorkflowCLIOptions struct {
	EngineDB   string
	WorkflowID string
	BaseURL    string
	MaxCycles  int
	Fixture    bool
}

type BuildHTTPWorkflow func(baseURL string, workflowID string) (storecontract.CreateWorkflowParams, model.OpID, error)

type HTTPWorkflowSpec struct {
	Site           model.SiteName
	Entrypoint     string
	DefaultBaseURL string
	FixtureName    string
	BuildWorkflow  BuildHTTPWorkflow
	RegisterSite   func(*siteregistry.Registry) error
	LoadFixture    func(string) ([]byte, error)
}

func AddSharedHTTPWorkflowFlags(cmd *cobra.Command, options *HTTPWorkflowCLIOptions, defaultBaseURL string) {
	cmd.Flags().StringVar(
		&options.EngineDB,
		"engine-db",
		"state/engine.db",
		"Path to the durable engine SQLite database",
	)
	cmd.Flags().StringVar(
		&options.WorkflowID,
		"workflow-id",
		"",
		"Workflow ID to use for the run (defaults to a timestamped ID)",
	)
	cmd.Flags().StringVar(
		&options.BaseURL,
		"base-url",
		defaultBaseURL,
		"Base URL used by the HTTP-backed workflow",
	)
	cmd.Flags().IntVar(
		&options.MaxCycles,
		"max-cycles",
		32,
		"Maximum scheduler cycles before the command gives up",
	)
	cmd.Flags().BoolVar(
		&options.Fixture,
		"fixture",
		false,
		"Serve the embedded frontpage fixture from a temporary local HTTP server instead of using --base-url",
	)
}

func RunHTTPWorkflowCommand(cmd *cobra.Command, options *HTTPWorkflowCLIOptions, spec HTTPWorkflowSpec) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if options == nil {
		options = &HTTPWorkflowCLIOptions{}
	}
	if spec.RegisterSite == nil {
		return fmt.Errorf("site %s does not provide a registry hook", spec.Site)
	}
	if spec.BuildWorkflow == nil {
		return fmt.Errorf("site %s entrypoint %s does not provide a workflow builder", spec.Site, spec.Entrypoint)
	}

	registry := siteregistry.New()
	if err := spec.RegisterSite(registry); err != nil {
		return err
	}

	manager := sitemigrate.NewManager(registry)
	report, err := manager.Migrate(ctx, spec.Site, cmd.Flag("sites-dir").Value.String())
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(options.EngineDB), 0o755); err != nil {
		return fmt.Errorf("create engine db directory: %w", err)
	}

	engineStore, err := sqlitestore.Open(ctx, options.EngineDB)
	if err != nil {
		return err
	}
	defer func() { _ = engineStore.Close() }()

	siteDB, err := sql.Open("sqlite3", report.DatabasePath)
	if err != nil {
		return fmt.Errorf("open site db: %w", err)
	}
	defer func() { _ = siteDB.Close() }()

	baseURL, client, closer, err := resolveHTTPBaseURL(options, spec)
	if err != nil {
		return err
	}
	if closer != nil {
		defer func() { _ = closer.Close() }()
	}

	runners := runner.NewRegistry()
	if err := runners.Register(runner.NewHTTPRunner(config.HTTP{
		UserAgent: "scraper/site-runner",
		Timeout:   15 * time.Second,
	}, client)); err != nil {
		return err
	}
	if err := runners.Register(runner.NewJSRunner(registry)); err != nil {
		return err
	}

	s, err := scheduler.New(engineStore, runners, scheduler.Config{
		MaxWorkers:           4,
		PollInterval:         25 * time.Millisecond,
		DefaultLeaseDuration: 30 * time.Second,
	}, fmt.Sprintf("%s-cli", spec.Site), nil)
	if err != nil {
		return err
	}
	s.SetSiteDBProvider(func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error) {
		if site != spec.Site {
			return nil, fmt.Errorf("unexpected site db request for %s", site)
		}
		return siteDB, nil
	})

	workflowParams, targetOpID, err := spec.BuildWorkflow(baseURL, options.WorkflowID)
	if err != nil {
		return err
	}

	if err := s.CreateWorkflow(ctx, workflowParams); err != nil {
		return err
	}

	maxCycles := options.MaxCycles
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
		return fmt.Errorf("%s result %s not found", spec.Entrypoint, targetOpID)
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

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Site: %s\n", spec.Site)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Entrypoint: %s\n", spec.Entrypoint)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Workflow: %s\n", workflow.ID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", workflow.Status)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Engine DB: %s\n", options.EngineDB)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Site DB: %s\n", report.DatabasePath)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Base URL: %s\n", baseURL)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Fixture: %t\n", options.Fixture)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Result op: %s\n", targetOpID)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Result:\n%s\n", prettySummary)

	return nil
}

func resolveHTTPBaseURL(options *HTTPWorkflowCLIOptions, spec HTTPWorkflowSpec) (string, *http.Client, io.Closer, error) {
	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = spec.DefaultBaseURL
	}
	if !options.Fixture {
		return baseURL, nil, nil, nil
	}
	if spec.LoadFixture == nil {
		return "", nil, nil, fmt.Errorf("site %s does not provide fixture loading", spec.Site)
	}

	fixtureName := spec.FixtureName
	if fixtureName == "" {
		fixtureName = "frontpage.html"
	}
	body, err := spec.LoadFixture(fixtureName)
	if err != nil {
		return "", nil, nil, fmt.Errorf("load fixture %s: %w", fixtureName, err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(body)
	}))
	return server.URL + "/", server.Client(), closeFunc(func() error {
		server.Close()
		return nil
	}), nil
}

type closeFunc func() error

func (f closeFunc) Close() error {
	return f()
}

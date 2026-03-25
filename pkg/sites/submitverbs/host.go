package submitverbs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/scraper/pkg/engine/config"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	sitemigrate "github.com/go-go-golems/scraper/pkg/sites/migrate"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
)

type Host struct {
	siteRegistry *siteregistry.Registry
	def          siteregistry.Definition
	registry     *jsverbs.Registry
}

type SubmitOptions struct {
	EngineDB   string
	SitesDir   string
	WorkflowID string
}

type SubmitResult struct {
	Workflow    model.WorkflowRun
	SiteDBPath  string
	TargetOpID  model.OpID
	Submitted   []model.OpSpec
	VerbData    json.RawMessage
	CommandPath string
}

func NewHost(siteRegistry *siteregistry.Registry, def siteregistry.Definition, registry *jsverbs.Registry) *Host {
	return &Host{
		siteRegistry: siteRegistry,
		def:          def,
		registry:     registry,
	}
}

func (h *Host) Submit(
	ctx context.Context,
	verb *jsverbs.VerbSpec,
	parsedValues *values.Values,
	options SubmitOptions,
) (*SubmitResult, error) {
	if h == nil {
		return nil, fmt.Errorf("submit host is nil")
	}
	if verb == nil {
		return nil, fmt.Errorf("verb is required")
	}
	if options.EngineDB == "" {
		options.EngineDB = "state/engine.db"
	}
	if options.SitesDir == "" {
		options.SitesDir = "state/sites"
	}

	engineStore, err := openEngineStore(ctx, options.EngineDB)
	if err != nil {
		return nil, err
	}
	defer func() { _ = engineStore.Close() }()

	scraperDB, err := openScraperDB(options.EngineDB)
	if err != nil {
		return nil, err
	}
	defer func() { _ = scraperDB.Close() }()

	siteProvider := newSiteDBProvider(h.siteRegistry, options.SitesDir)
	defer func() { _ = siteProvider.Close() }()

	siteReport, err := siteProvider.manager.Migrate(ctx, h.def.Name, options.SitesDir)
	if err != nil {
		return nil, err
	}
	siteDB, err := siteProvider.QueryExecer(ctx, h.def.Name)
	if err != nil {
		return nil, err
	}

	workflow, err := newWorkflowForVerb(h.def.Name, verb, parsedValues, options.WorkflowID)
	if err != nil {
		return nil, err
	}

	executor := NewExecutor(ExecutorConfig{
		Registry:                h.registry,
		VerbsFS:                 h.def.VerbsFS,
		VerbsRoot:               h.def.VerbsRoot,
		Modules:                 h.def.Modules,
		RuntimeModuleRegistrars: h.def.RuntimeModuleRegistrars,
		ScraperDB:               scraperDB,
		SiteDB:                  siteDB,
	})
	execResult, err := executor.Execute(ctx, ExecutionRequest{
		Site:         h.def.Name,
		Verb:         verb,
		ParsedValues: parsedValues,
		Workflow:     workflow,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}

	runners, err := newDefaultRunnerRegistry(h.siteRegistry, config.HTTP{
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	s, err := scheduler.New(engineStore, runners, scheduler.Config{
		MaxWorkers:           1,
		PollInterval:         50 * time.Millisecond,
		DefaultLeaseDuration: 30 * time.Second,
	}, "submit-"+string(h.def.Name), nil)
	if err != nil {
		return nil, err
	}

	if err := s.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: execResult.Workflow,
		Initial:  execResult.Submitted,
	}); err != nil {
		return nil, err
	}

	return &SubmitResult{
		Workflow:   execResult.Workflow,
		SiteDBPath: siteReport.DatabasePath,
		TargetOpID: execResult.TargetOpID,
		Submitted:  execResult.Submitted,
		VerbData:   execResult.VerbData,
	}, nil
}

func PrintSubmitResult(w io.Writer, result *SubmitResult, engineDB string) error {
	if result == nil {
		return fmt.Errorf("submit result is nil")
	}

	prettyData := "{}"
	if len(result.VerbData) > 0 {
		var decoded any
		if err := json.Unmarshal(result.VerbData, &decoded); err == nil {
			if pretty, err := json.MarshalIndent(decoded, "", "  "); err == nil {
				prettyData = string(pretty)
			}
		}
	}

	_, _ = fmt.Fprintf(w, "Site: %s\n", result.Workflow.Site)
	_, _ = fmt.Fprintf(w, "Command: %s\n", result.CommandPath)
	_, _ = fmt.Fprintf(w, "Workflow: %s\n", result.Workflow.ID)
	_, _ = fmt.Fprintf(w, "Engine DB: %s\n", engineDB)
	_, _ = fmt.Fprintf(w, "Site DB: %s\n", result.SiteDBPath)
	_, _ = fmt.Fprintf(w, "Submitted ops: %d\n", len(result.Submitted))
	if result.TargetOpID != "" {
		_, _ = fmt.Fprintf(w, "Target op: %s\n", result.TargetOpID)
	}
	_, _ = fmt.Fprintf(w, "Verb result:\n%s\n", prettyData)
	return nil
}

func openEngineStore(ctx context.Context, engineDB string) (*sqlitestore.Store, error) {
	if err := ensureEngineDBDir(engineDB); err != nil {
		return nil, err
	}
	return sqlitestore.Open(ctx, engineDB)
}

func openScraperDB(engineDB string) (*sql.DB, error) {
	if err := ensureEngineDBDir(engineDB); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", engineDB)
	if err != nil {
		return nil, fmt.Errorf("open scraper db: %w", err)
	}
	return db, nil
}

func newDefaultRunnerRegistry(siteRegistry *siteregistry.Registry, httpConfig config.HTTP) (*runner.Registry, error) {
	runners := runner.NewRegistry()
	httpRunner, err := runner.NewHTTPRunner(httpConfig, nil)
	if err != nil {
		return nil, err
	}
	if err := runners.Register(httpRunner); err != nil {
		return nil, err
	}
	if err := runners.Register(runner.NewJSRunner(siteRegistry)); err != nil {
		return nil, err
	}
	return runners, nil
}

func newSiteDBProvider(siteRegistry *siteregistry.Registry, sitesDir string) *siteDBProvider {
	return &siteDBProvider{
		manager:  sitemigrate.NewManager(siteRegistry),
		sitesDir: sitesDir,
		dbs:      map[model.SiteName]*sql.DB{},
	}
}

type siteDBProvider struct {
	manager  *sitemigrate.Manager
	sitesDir string
	dbs      map[model.SiteName]*sql.DB
}

func (p *siteDBProvider) QueryExecer(ctx context.Context, site model.SiteName) (*sql.DB, error) {
	if db, ok := p.dbs[site]; ok {
		return db, nil
	}

	report, err := p.manager.Migrate(ctx, site, p.sitesDir)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", report.DatabasePath)
	if err != nil {
		return nil, fmt.Errorf("open site db: %w", err)
	}
	p.dbs[site] = db
	return db, nil
}

func (p *siteDBProvider) Close() error {
	var firstErr error
	for site, db := range p.dbs {
		if err := db.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close site db %s: %w", site, err)
		}
		delete(p.dbs, site)
	}
	return firstErr
}

func ensureEngineDBDir(engineDB string) error {
	dir := filepath.Dir(engineDB)
	if dir == "." || dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create engine db directory: %w", err)
	}
	return nil
}

func newWorkflowForVerb(
	site model.SiteName,
	verb *jsverbs.VerbSpec,
	parsedValues *values.Values,
	overrideID string,
) (model.WorkflowRun, error) {
	if parsedValues == nil {
		parsedValues = values.New()
	}

	input, err := json.Marshal(parsedValues.GetDataMap())
	if err != nil {
		return model.WorkflowRun{}, fmt.Errorf("marshal workflow input: %w", err)
	}

	workflowID := overrideID
	if workflowID == "" {
		workflowID = fmt.Sprintf("%s-%s-%d", site, verb.Name, time.Now().UTC().UnixNano())
	}

	return model.WorkflowRun{
		ID:     model.WorkflowID(workflowID),
		Site:   site,
		Name:   fmt.Sprintf("%s %s submission", site, verb.Name),
		Status: model.WorkflowStatusPending,
		Input:  input,
		Metadata: map[string]string{
			"submitVerb": verb.FullPath(),
		},
	}, nil
}

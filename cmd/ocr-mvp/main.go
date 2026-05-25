package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/workflow"
	"github.com/go-go-golems/scraper/pkg/workflows/ocrmvp"
	"github.com/go-go-golems/scraper/pkg/workflows/ocrquality"
)

const defaultWorkDir = ".ocr-mvp"

type registryFlags []string

func (f *registryFlags) String() string { return strings.Join(*f, ",") }
func (f *registryFlags) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			*f = append(*f, trimmed)
		}
	}
	return nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return runWorkflow(args)
	}
	subcommand := args[0]
	subArgs := args[1:]
	switch subcommand {
	case "run":
		return runWorkflow(subArgs)
	case "retry":
		return retryStep(subArgs)
	case "cancel":
		return cancelRun(subArgs)
	case "status":
		return showStatus(subArgs)
	case "pages":
		return listPages(subArgs)
	case "quality-pass":
		return runQualityPass(subArgs)
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown subcommand %q", subcommand)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage:
  ocr-mvp run [flags]
  ocr-mvp [run flags]              # backwards-compatible shorthand for run
  ocr-mvp status --work-dir DIR --run-id RUN_ID
  ocr-mvp retry --work-dir DIR --run-id RUN_ID --step-id STEP_ID
  ocr-mvp cancel --work-dir DIR --run-id RUN_ID
  ocr-mvp pages --work-dir DIR --book-id BOOK_ID [--status STATUS]
  ocr-mvp quality-pass --markdown PATH --output-dir DIR [--expected-pages N]

Run flags include --book-id, --image-dir, --work-dir, --profile, --profile-registries, --prompt-version, --context-window, --log-level, --dry-run, and --max-workers.
`)
}

func runWorkflow(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	var registries registryFlags
	bookID := fs.String("book-id", "", "Book identifier for workflow metadata and output names")
	imageDir := fs.String("image-dir", "", "Directory containing page images")
	pageGlob := fs.String("page-glob", "page_*.png", "Glob used inside image-dir to discover page images")
	startPage := fs.Int("start-page", 0, "Optional first page number to process")
	endPage := fs.Int("end-page", 0, "Optional last page number to process")
	workDir := fs.String("work-dir", defaultWorkDir, "Directory for engine DB, artifacts, and projections")
	profile := fs.String("profile", "", "Optional Pinocchio profile slug; empty uses Pinocchio defaults")
	promptVersion := fs.String("prompt-version", ocrmvp.DefaultPromptVersion, "OCR prompt version to use")
	contextWindow := fs.Int("context-window", 0, "Optional number of previous/next page images to include as OCR continuity context (0-2)")
	logLevel := fs.String("log-level", "info", "zerolog level: trace, debug, info, warn, error, disabled")
	dryRun := fs.Bool("dry-run", false, "Use deterministic dry-run OCR instead of live Geppetto inference")
	maxWorkers := fs.Int("max-workers", 4, "Maximum concurrent workflow workers")
	pollInterval := fs.Duration("poll-interval", 250*time.Millisecond, "Worker polling interval")
	runID := fs.String("run-id", "", "Optional stable workflow run ID")
	fs.Var(&registries, "profile-registries", "Pinocchio profile registry source (repeatable or comma-separated)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := configureLogLevel(*logLevel); err != nil {
		return err
	}

	if strings.TrimSpace(*bookID) == "" {
		return fmt.Errorf("--book-id is required")
	}
	if strings.TrimSpace(*imageDir) == "" {
		return fmt.Errorf("--image-dir is required")
	}
	absImageDir, err := filepath.Abs(*imageDir)
	if err != nil {
		return err
	}
	paths, err := resolveWorkDir(*workDir, true)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := ocrmvp.OCRClient(ocrmvp.NewGeppettoOCRClient())
	if *dryRun {
		client = ocrmvp.DryRunOCRClient{}
	}
	rt, err := newRuntime(ctx, paths, *maxWorkers, *pollInterval)
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close() }()
	if err := ocrmvp.Register(rt, ocrmvp.Config{Client: client}); err != nil {
		return err
	}

	runOpts := []workflow.RunOption{workflow.WithRunName("OCR MVP: " + *bookID)}
	if strings.TrimSpace(*runID) != "" {
		runOpts = append(runOpts, workflow.WithRunID(*runID))
	}
	handle, err := rt.StartRun(ctx, ocrmvp.PackageName, ocrmvp.RunInput{
		BookID:            *bookID,
		ImageDir:          absImageDir,
		PageGlob:          *pageGlob,
		StartPage:         *startPage,
		EndPage:           *endPage,
		Profile:           *profile,
		ProfileRegistries: append([]string(nil), registries...),
		PromptVersion:     *promptVersion,
		ContextWindow:     *contextWindow,
		DryRun:            *dryRun,
	}, runOpts...)
	if err != nil {
		return err
	}
	fmt.Printf("started run %s in %s\n", handle.ID, paths.root)

	for {
		cycle, err := rt.RunOnce(ctx)
		if err != nil {
			return err
		}
		wf, err := rt.Workflow(ctx, handle.ID)
		if err != nil {
			return err
		}
		fmt.Printf("status=%s processed=%d succeeded=%d failed=%d retried=%d\n", wf.Status, cycle.Processed, cycle.Succeeded, cycle.Failed, cycle.Retried)
		if isTerminal(wf.Status) {
			if wf.Status != model.WorkflowStatusSucceeded {
				return fmt.Errorf("workflow finished with status %s", wf.Status)
			}
			result, err := rt.Result(ctx, handle.ID, "assemble-markdown")
			if err == nil && result != nil {
				fmt.Printf("assemble result: %s\n", string(result.Data))
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(*pollInterval):
		}
	}
}

func runQualityPass(args []string) error {
	fs := flag.NewFlagSet("quality-pass", flag.ContinueOnError)
	markdownPath := fs.String("markdown", "", "OCR markdown file to QA and normalize")
	outputDir := fs.String("output-dir", "", "Directory for normalized markdown and diff")
	workDir := fs.String("work-dir", defaultWorkDir, "Directory for engine DB, artifacts, and projections")
	bookID := fs.String("book-id", "", "Optional book identifier for report metadata")
	bookProfile := fs.String("book-profile", "", "Optional YAML book profile; workflow writes discoveries separately and never mutates this file")
	discoveryPath := fs.String("discovery", "", "Optional output path for machine-updated book discovery YAML")
	profilePatchPath := fs.String("profile-patch", "", "Optional output path for proposed profile patch YAML")
	expectedPages := fs.Int("expected-pages", 0, "Expected page marker count; 0 uses book profile/default")
	logPath := fs.String("log", "", "Optional OCR run log to import into SQLite")
	imageDir := fs.String("image-dir", "", "Optional page image directory for embedded figure extraction")
	embedFigures := fs.Bool("embed-figures", false, "Extract figure images from source pages and embed markdown image links")
	maxWorkers := fs.Int("max-workers", 2, "Maximum concurrent workflow workers")
	pollInterval := fs.Duration("poll-interval", 250*time.Millisecond, "Worker polling interval")
	runID := fs.String("run-id", "", "Optional stable workflow run ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*markdownPath) == "" {
		return fmt.Errorf("--markdown is required")
	}
	absMarkdown, err := filepath.Abs(*markdownPath)
	if err != nil {
		return err
	}
	outDir := strings.TrimSpace(*outputDir)
	if outDir == "" {
		outDir = filepath.Join(filepath.Dir(absMarkdown), "quality-pass")
	}
	absOutputDir, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}
	paths, err := resolveWorkDir(*workDir, true)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	rt, err := newRuntime(ctx, paths, *maxWorkers, *pollInterval)
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close() }()
	if err := ocrquality.Register(rt); err != nil {
		return err
	}
	runOpts := []workflow.RunOption{workflow.WithRunName("OCR Quality Pass")}
	if strings.TrimSpace(*runID) != "" {
		runOpts = append(runOpts, workflow.WithRunID(*runID))
	}
	handle, err := rt.StartRun(ctx, ocrquality.PackageName, ocrquality.RunInput{BookID: *bookID, BookProfilePath: *bookProfile, DiscoveryPath: *discoveryPath, ProfilePatchPath: *profilePatchPath, MarkdownPath: absMarkdown, OutputDir: absOutputDir, ExpectedPages: *expectedPages, LogPath: *logPath, ImageDir: *imageDir, EmbedFigures: *embedFigures}, runOpts...)
	if err != nil {
		return err
	}
	fmt.Printf("started quality run %s in %s\n", handle.ID, paths.root)
	for {
		cycle, err := rt.RunOnce(ctx)
		if err != nil {
			return err
		}
		wf, err := rt.Workflow(ctx, handle.ID)
		if err != nil {
			return err
		}
		fmt.Printf("status=%s processed=%d succeeded=%d failed=%d retried=%d\n", wf.Status, cycle.Processed, cycle.Succeeded, cycle.Failed, cycle.Retried)
		if isTerminal(wf.Status) {
			if wf.Status != model.WorkflowStatusSucceeded {
				return fmt.Errorf("workflow finished with status %s", wf.Status)
			}
			fmt.Printf("quality output: %s\n", filepath.Join(absOutputDir, "normalized.md"))
			if *embedFigures {
				fmt.Printf("quality embedded output: %s\n", filepath.Join(absOutputDir, "embedded-figures.md"))
			}
			fmt.Printf("quality diff: %s\n", filepath.Join(absOutputDir, "cleanup.diff"))
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(*pollInterval):
		}
	}
}

func retryStep(args []string) error {
	fs := flag.NewFlagSet("retry", flag.ContinueOnError)
	workDir := fs.String("work-dir", defaultWorkDir, "Directory containing engine.db")
	runID := fs.String("run-id", "", "Workflow run ID")
	stepID := fs.String("step-id", "", "Failed step/op ID to retry")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*runID) == "" {
		return fmt.Errorf("--run-id is required")
	}
	if strings.TrimSpace(*stepID) == "" {
		return fmt.Errorf("--step-id is required")
	}
	ctx := context.Background()
	rt, err := openExistingRuntime(ctx, *workDir)
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close() }()
	if err := rt.RetryStep(ctx, model.WorkflowID(*runID), model.OpID(*stepID)); err != nil {
		return err
	}
	fmt.Printf("retried step %s in run %s\n", *stepID, *runID)
	return nil
}

func cancelRun(args []string) error {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	workDir := fs.String("work-dir", defaultWorkDir, "Directory containing engine.db")
	runID := fs.String("run-id", "", "Workflow run ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*runID) == "" {
		return fmt.Errorf("--run-id is required")
	}
	ctx := context.Background()
	rt, err := openExistingRuntime(ctx, *workDir)
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close() }()
	if err := rt.CancelRun(ctx, model.WorkflowID(*runID)); err != nil {
		return err
	}
	fmt.Printf("canceled run %s\n", *runID)
	return nil
}

func showStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	workDir := fs.String("work-dir", defaultWorkDir, "Directory containing engine.db")
	runID := fs.String("run-id", "", "Workflow run ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*runID) == "" {
		return fmt.Errorf("--run-id is required")
	}
	ctx := context.Background()
	rt, err := openExistingRuntime(ctx, *workDir)
	if err != nil {
		return err
	}
	defer func() { _ = rt.Close() }()
	wf, err := rt.Workflow(ctx, model.WorkflowID(*runID))
	if err != nil {
		return err
	}
	fmt.Printf("run_id=%s\nsite=%s\nname=%s\nstatus=%s\ncreated_at=%s\nupdated_at=%s\n", wf.ID, wf.Site, wf.Name, wf.Status, wf.CreatedAt.Format(time.RFC3339), wf.UpdatedAt.Format(time.RFC3339))
	return nil
}

func listPages(args []string) error {
	fs := flag.NewFlagSet("pages", flag.ContinueOnError)
	workDir := fs.String("work-dir", defaultWorkDir, "Directory containing projections")
	bookID := fs.String("book-id", "", "Book ID to list")
	status := fs.String("status", "", "Optional page status filter")
	limit := fs.Int("limit", 0, "Optional maximum number of rows")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*bookID) == "" {
		return fmt.Errorf("--book-id is required")
	}
	paths, err := resolveWorkDir(*workDir, false)
	if err != nil {
		return err
	}
	ctx := context.Background()
	store := workflow.NewSQLiteProjectionStore(paths.projectionsDir)
	defer func() { _ = store.Close() }()
	projection, err := store.Projection(ctx, ocrmvp.PackageName)
	if err != nil {
		return err
	}
	if err := ocrmvp.EnsureProjectionSchema(ctx, projection); err != nil {
		return err
	}
	query := `SELECT page_num, status, ocr_step_id, char_count, error_code, markdown_artifact_uri, updated_at FROM pages WHERE book_id = ?`
	queryArgs := []any{*bookID}
	if strings.TrimSpace(*status) != "" {
		query += ` AND status = ?`
		queryArgs = append(queryArgs, *status)
	}
	query += ` ORDER BY page_num`
	if *limit > 0 {
		query += ` LIMIT ?`
		queryArgs = append(queryArgs, *limit)
	}
	rows, err := projection.Query(ctx, query, queryArgs...)
	if err != nil {
		return err
	}
	printPageRows(rows)
	return nil
}

type workPaths struct {
	root           string
	engineDB       string
	artifactsDir   string
	projectionsDir string
}

func resolveWorkDir(workDir string, create bool) (workPaths, error) {
	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return workPaths{}, err
	}
	if create {
		if err := os.MkdirAll(absWorkDir, 0o755); err != nil {
			return workPaths{}, err
		}
	}
	return workPaths{
		root:           absWorkDir,
		engineDB:       filepath.Join(absWorkDir, "engine.db"),
		artifactsDir:   filepath.Join(absWorkDir, "artifacts"),
		projectionsDir: filepath.Join(absWorkDir, "projections"),
	}, nil
}

func openExistingRuntime(ctx context.Context, workDir string) (*workflow.Runtime, error) {
	paths, err := resolveWorkDir(workDir, false)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(paths.engineDB); err != nil {
		return nil, fmt.Errorf("open engine DB %s: %w", paths.engineDB, err)
	}
	return newRuntime(ctx, paths, 1, 250*time.Millisecond)
}

func newRuntime(ctx context.Context, paths workPaths, maxWorkers int, pollInterval time.Duration) (*workflow.Runtime, error) {
	return workflow.NewRuntime(ctx, workflow.Config{
		Store:           workflow.SQLiteStore(paths.engineDB),
		ArtifactStore:   workflow.NewFileArtifactStore(paths.artifactsDir),
		ProjectionStore: workflow.NewSQLiteProjectionStore(paths.projectionsDir),
		WorkerID:        "ocr-mvp-cli",
		MaxWorkers:      maxWorkers,
		PollInterval:    pollInterval,
		Queues: map[model.QueueKey]workflow.QueueConfig{
			ocrmvp.QueueControl:     {MaxWorkers: 1},
			ocrmvp.QueueOCR:         {MaxWorkers: maxWorkers},
			ocrmvp.QueueAssemble:    {MaxWorkers: 1},
			ocrquality.QueueQuality: {MaxWorkers: maxWorkers},
		},
	})
}

func configureLogLevel(value string) error {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		value = "info"
	}
	if value == "disabled" || value == "off" || value == "none" {
		zerolog.SetGlobalLevel(zerolog.Disabled)
		return nil
	}
	level, err := zerolog.ParseLevel(value)
	if err != nil {
		return fmt.Errorf("invalid --log-level %q: %w", value, err)
	}
	zerolog.SetGlobalLevel(level)
	return nil
}

func printPageRows(rows []map[string]any) {
	if len(rows) == 0 {
		fmt.Println("no pages")
		return
	}
	for _, row := range rows {
		keys := make([]string, 0, len(row))
		for key := range row {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%s=%v", key, row[key]))
		}
		fmt.Println(strings.Join(parts, " "))
	}
}

func isTerminal(status model.WorkflowStatus) bool {
	switch status {
	case model.WorkflowStatusSucceeded, model.WorkflowStatusFailed, model.WorkflowStatusCanceled:
		return true
	case model.WorkflowStatusPending, model.WorkflowStatusRunning:
		return false
	default:
		return false
	}
}

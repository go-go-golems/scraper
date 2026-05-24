package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/workflow"
	"github.com/go-go-golems/scraper/pkg/workflows/ocrmvp"
)

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
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var registries registryFlags
	bookID := flag.String("book-id", "", "Book identifier for workflow metadata and output names")
	imageDir := flag.String("image-dir", "", "Directory containing page images")
	pageGlob := flag.String("page-glob", "page_*.png", "Glob used inside image-dir to discover page images")
	startPage := flag.Int("start-page", 0, "Optional first page number to process")
	endPage := flag.Int("end-page", 0, "Optional last page number to process")
	workDir := flag.String("work-dir", ".ocr-mvp", "Directory for engine DB, artifacts, and projections")
	profile := flag.String("profile", "", "Optional Pinocchio profile slug; empty uses Pinocchio defaults")
	dryRun := flag.Bool("dry-run", false, "Use deterministic dry-run OCR instead of live Geppetto inference")
	maxWorkers := flag.Int("max-workers", 4, "Maximum concurrent workflow workers")
	pollInterval := flag.Duration("poll-interval", 250*time.Millisecond, "Worker polling interval")
	runID := flag.String("run-id", "", "Optional stable workflow run ID")
	flag.Var(&registries, "profile-registries", "Pinocchio profile registry source (repeatable or comma-separated)")
	flag.Parse()

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
	absWorkDir, err := filepath.Abs(*workDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(absWorkDir, 0o755); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := ocrmvp.OCRClient(ocrmvp.NewGeppettoOCRClient())
	if *dryRun {
		client = ocrmvp.DryRunOCRClient{}
	}
	rt, err := workflow.NewRuntime(ctx, workflow.Config{
		Store:           workflow.SQLiteStore(filepath.Join(absWorkDir, "engine.db")),
		ArtifactStore:   workflow.NewFileArtifactStore(filepath.Join(absWorkDir, "artifacts")),
		ProjectionStore: workflow.NewSQLiteProjectionStore(filepath.Join(absWorkDir, "projections")),
		WorkerID:        "ocr-mvp-cli",
		MaxWorkers:      *maxWorkers,
		PollInterval:    *pollInterval,
		Queues: map[model.QueueKey]workflow.QueueConfig{
			ocrmvp.QueueControl:  {MaxWorkers: 1},
			ocrmvp.QueueOCR:      {MaxWorkers: *maxWorkers},
			ocrmvp.QueueAssemble: {MaxWorkers: 1},
		},
	})
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
		PromptVersion:     ocrmvp.DefaultPromptVersion,
		DryRun:            *dryRun,
	}, runOpts...)
	if err != nil {
		return err
	}
	fmt.Printf("started run %s in %s\n", handle.ID, absWorkDir)

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

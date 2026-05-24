package ocrmvp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/workflow"
)

type Config struct {
	Client         OCRClient
	ProjectionName string
}

func Register(rt *workflow.Runtime, cfg Config) error {
	if rt == nil {
		return fmt.Errorf("workflow runtime is nil")
	}
	if cfg.Client == nil {
		cfg.Client = DryRunOCRClient{}
	}
	if cfg.ProjectionName == "" {
		cfg.ProjectionName = ProjectionName
	}
	if err := rt.RegisterPackage(Package()); err != nil {
		return err
	}
	for _, executor := range []workflow.Executor{
		DiscoverPagesExecutor(cfg.ProjectionName),
		OCRPageExecutor(cfg.ProjectionName, cfg.Client),
		AssembleMarkdownExecutor(cfg.ProjectionName),
	} {
		if err := rt.RegisterExecutor(executor); err != nil {
			return err
		}
	}
	return nil
}

func Package() *workflow.Package {
	return workflow.NewPackage(PackageName).
		DisplayName("MVP OCR Workflow").
		Entrypoint(workflow.EntrypointFunc[RunInput](func(ctx context.Context, run *workflow.RunBuilder, input RunInput) error {
			input = normalizeRunInput(input)
			if err := validateRunInput(input); err != nil {
				return err
			}
			run.Metadata("book_id", input.BookID)
			run.Metadata("image_dir", input.ImageDir)
			run.Metadata("prompt_version", input.PromptVersion)
			_, err := run.Step("discover-pages", input, workflow.StepOpts{
				Kind:  KindDiscoverPages,
				Queue: QueueControl,
				Retry: model.RetryPolicy{MaxAttempts: 1},
			})
			return err
		})).
		Build()
}

func normalizeRunInput(input RunInput) RunInput {
	input.BookID = strings.TrimSpace(input.BookID)
	input.ImageDir = strings.TrimSpace(input.ImageDir)
	input.PageGlob = strings.TrimSpace(input.PageGlob)
	if input.PageGlob == "" {
		input.PageGlob = "page_*.png"
	}
	input.Profile = strings.TrimSpace(input.Profile)
	input.PromptVersion = normalizePromptVersion(strings.TrimSpace(input.PromptVersion))
	return input
}

func validateRunInput(input RunInput) error {
	if input.BookID == "" {
		return fmt.Errorf("book_id is required")
	}
	if input.ImageDir == "" {
		return fmt.Errorf("image_dir is required")
	}
	if input.StartPage > 0 && input.EndPage > 0 && input.StartPage > input.EndPage {
		return fmt.Errorf("start_page must be <= end_page")
	}
	return nil
}

func ocrRetryPolicy() model.RetryPolicy {
	return model.RetryPolicy{
		MaxAttempts:    3,
		BackoffKind:    model.BackoffKindExponential,
		InitialBackoff: time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2,
	}
}

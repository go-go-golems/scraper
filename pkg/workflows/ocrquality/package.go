package ocrquality

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/workflow"
)

func Register(rt *workflow.Runtime) error {
	if rt == nil {
		return fmt.Errorf("workflow runtime is nil")
	}
	if err := rt.RegisterPackage(Package()); err != nil {
		return err
	}
	for _, executor := range []workflow.Executor{
		QABeforeExecutor(),
		NormalizeExecutor(),
		QAAfterExecutor(),
		ImportLogExecutor(),
		EmbedFiguresExecutor(),
		AssembleReportExecutor(),
	} {
		if err := rt.RegisterExecutor(executor); err != nil {
			return err
		}
	}
	return nil
}

func Package() *workflow.Package {
	return workflow.NewPackage(PackageName).
		DisplayName("OCR Quality Pass").
		Entrypoint(workflow.EntrypointFunc[RunInput](func(ctx context.Context, run *workflow.RunBuilder, input RunInput) error {
			input = normalizeRunInput(input)
			if err := validateRunInput(input); err != nil {
				return err
			}
			run.Metadata("book_id", input.BookID)
			run.Metadata("markdown_path", input.MarkdownPath)
			before, err := run.Step("qa-before", QAInput{
				MarkdownPath:    input.MarkdownPath,
				ExpectedPages:   input.ExpectedPages,
				KnownBadTerms:   input.KnownBadTerms,
				ExpectedStrings: input.ExpectedStrings,
				ListPages:       input.ListPages,
				ReportName:      "qa-before.md",
			}, workflow.StepOpts{Kind: KindQABefore, Queue: QueueQuality, Retry: model.RetryPolicy{MaxAttempts: 1}})
			if err != nil {
				return err
			}
			normalizedPath := filepath.Join(input.OutputDir, "normalized.md")
			diffPath := filepath.Join(input.OutputDir, "cleanup.diff")
			normalized, err := run.Step("normalize-markdown", NormalizeInput{
				MarkdownPath: input.MarkdownPath,
				OutputPath:   normalizedPath,
				DiffPath:     diffPath,
				ListPages:    input.ListPages,
			}, workflow.StepOpts{Kind: KindNormalize, Queue: QueueQuality, DependsOn: workflow.Require(before), Retry: model.RetryPolicy{MaxAttempts: 1}})
			if err != nil {
				return err
			}
			qaAfterPath := normalizedPath
			qaAfterDepends := workflow.Require(normalized)
			embeddedPath := ""
			if input.EmbedFigures {
				embeddedPath = filepath.Join(input.OutputDir, "embedded-figures.md")
				embedded, err := run.Step("embed-figures", EmbedFiguresInput{
					MarkdownPath: normalizedPath,
					ImageDir:     input.ImageDir,
					OutputPath:   embeddedPath,
					FiguresDir:   filepath.Join(input.OutputDir, "figures"),
				}, workflow.StepOpts{Kind: KindEmbedFigures, Queue: QueueQuality, DependsOn: workflow.Require(normalized), Retry: model.RetryPolicy{MaxAttempts: 1}})
				if err != nil {
					return err
				}
				qaAfterPath = embeddedPath
				qaAfterDepends = workflow.Require(embedded)
			}
			after, err := run.Step("qa-after", QAInput{
				MarkdownPath:    qaAfterPath,
				ExpectedPages:   input.ExpectedPages,
				KnownBadTerms:   input.KnownBadTerms,
				ExpectedStrings: input.ExpectedStrings,
				ListPages:       input.ListPages,
				ReportName:      "qa-after.md",
			}, workflow.StepOpts{Kind: KindQAAfter, Queue: QueueQuality, DependsOn: qaAfterDepends, Retry: model.RetryPolicy{MaxAttempts: 1}})
			if err != nil {
				return err
			}
			var logStep workflow.StepHandle
			if strings.TrimSpace(input.LogPath) != "" {
				logStep, err = run.Step("import-log", LogImportInput{
					LogPath:    input.LogPath,
					SQLitePath: filepath.Join(input.OutputDir, "run-log.sqlite"),
					ReportName: "log-import.md",
				}, workflow.StepOpts{Kind: KindImportLog, Queue: QueueQuality, Retry: model.RetryPolicy{MaxAttempts: 1}})
				if err != nil {
					return err
				}
			}
			deps := workflow.Require(before, normalized, after)
			if logStep.ID != "" {
				deps = append(deps, workflow.Require(logStep)...)
			}
			_, err = run.Step("assemble-quality-report", ReportInput{BookID: input.BookID, RawMarkdownPath: input.MarkdownPath, NormalizedPath: normalizedPath, EmbeddedPath: embeddedPath}, workflow.StepOpts{Kind: KindAssembleReport, Queue: QueueQuality, DependsOn: deps, Retry: model.RetryPolicy{MaxAttempts: 1}})
			return err
		})).
		Build()
}

func normalizeRunInput(input RunInput) RunInput {
	input.BookID = strings.TrimSpace(input.BookID)
	input.MarkdownPath = strings.TrimSpace(input.MarkdownPath)
	input.OutputDir = strings.TrimSpace(input.OutputDir)
	if input.ExpectedPages == 0 {
		input.ExpectedPages = 30
	}
	if len(input.KnownBadTerms) == 0 {
		input.KnownBadTerms = defaultKnownBadTerms()
	}
	if len(input.ExpectedStrings) == 0 {
		input.ExpectedStrings = defaultExpectedStrings()
	}
	if len(input.ListPages) == 0 {
		input.ListPages = defaultListPages()
	}
	input.ImageDir = strings.TrimSpace(input.ImageDir)
	if input.EmbedFigures && input.ImageDir == "" {
		input.EmbedFigures = false
	}
	if input.OutputDir == "" && input.MarkdownPath != "" {
		input.OutputDir = filepath.Join(filepath.Dir(input.MarkdownPath), "quality-pass")
	}
	return input
}

func validateRunInput(input RunInput) error {
	if input.MarkdownPath == "" {
		return fmt.Errorf("markdown_path is required")
	}
	if input.OutputDir == "" {
		return fmt.Errorf("output_dir is required")
	}
	return nil
}

func QABeforeExecutor() workflow.Executor { return qaExecutor(KindQABefore) }
func QAAfterExecutor() workflow.Executor  { return qaExecutor(KindQAAfter) }

func qaExecutor(kind string) workflow.Executor {
	return workflow.NewTypedExecutor(kind, func(ctx context.Context, step *workflow.StepContext, input QAInput) error {
		body, err := os.ReadFile(input.MarkdownPath)
		if err != nil {
			return workflow.Permanent("ocr_quality_qa_read_failed", err)
		}
		result := AnalyzeMarkdown(string(body), input)
		name := strings.TrimSpace(input.ReportName)
		if name == "" {
			name = "qa-report.md"
		}
		ref, err := step.StoreArtifact(name, "text/markdown", []byte(result.ReportMarkdown), workflow.ArtifactKind("ocr-quality-qa-report"))
		if err != nil {
			return workflow.Retryable("ocr_quality_qa_artifact_failed", err)
		}
		result.ReportRefID = ref.ID
		result.ReportRefURI = ref.URI
		return step.Result(result)
	})
}

func NormalizeExecutor() workflow.Executor {
	return workflow.NewTypedExecutor(KindNormalize, func(ctx context.Context, step *workflow.StepContext, input NormalizeInput) error {
		body, err := os.ReadFile(input.MarkdownPath)
		if err != nil {
			return workflow.Permanent("ocr_quality_normalize_read_failed", err)
		}
		normalized, stats := NormalizeMarkdown(string(body), NormalizeOptions{ListPages: input.ListPages})
		if strings.TrimSpace(input.OutputPath) != "" {
			if err := os.MkdirAll(filepath.Dir(input.OutputPath), 0o755); err != nil {
				return workflow.Retryable("ocr_quality_normalize_output_dir_failed", err)
			}
			// #nosec G703 -- output path is explicit operator/workflow input for local artifact export.
			if err := os.WriteFile(input.OutputPath, []byte(normalized), 0o644); err != nil {
				return workflow.Retryable("ocr_quality_normalize_write_failed", err)
			}
		}
		diff := UnifiedLineDiff(input.MarkdownPath, input.OutputPath, string(body), normalized)
		if strings.TrimSpace(input.DiffPath) != "" {
			if err := os.MkdirAll(filepath.Dir(input.DiffPath), 0o755); err != nil {
				return workflow.Retryable("ocr_quality_diff_output_dir_failed", err)
			}
			// #nosec G703 -- diff path is explicit operator/workflow input for local artifact export.
			if err := os.WriteFile(input.DiffPath, []byte(diff), 0o644); err != nil {
				return workflow.Retryable("ocr_quality_diff_write_failed", err)
			}
		}
		outRef, err := step.StoreArtifact("normalized.md", "text/markdown", []byte(normalized), workflow.ArtifactKind("ocr-quality-normalized-markdown"))
		if err != nil {
			return workflow.Retryable("ocr_quality_normalize_artifact_failed", err)
		}
		diffRef, err := step.StoreArtifact("cleanup.diff", "text/x-diff", []byte(diff), workflow.ArtifactKind("ocr-quality-cleanup-diff"))
		if err != nil {
			return workflow.Retryable("ocr_quality_diff_artifact_failed", err)
		}
		return step.Result(NormalizeResult{InputPath: input.MarkdownPath, OutputPath: input.OutputPath, DiffPath: input.DiffPath, Changed: stats.Changed, ChangedLines: stats.ChangedLines, ChangedPages: stats.ChangedPages, OutputRefID: outRef.ID, OutputRefURI: outRef.URI, DiffRefID: diffRef.ID, DiffRefURI: diffRef.URI, NormalizedBytes: len(normalized)})
	})
}

func EmbedFiguresExecutor() workflow.Executor {
	return workflow.NewTypedExecutor(KindEmbedFigures, func(ctx context.Context, step *workflow.StepContext, input EmbedFiguresInput) error {
		body, err := os.ReadFile(input.MarkdownPath)
		if err != nil {
			return workflow.Permanent("ocr_quality_embed_read_failed", err)
		}
		embedded, figures, err := EmbedExtractedFigures(string(body), FigureExtractionOptions{ImageDir: input.ImageDir, OutputDir: input.FiguresDir})
		if err != nil {
			return workflow.Permanent("ocr_quality_embed_extract_failed", err)
		}
		if err := os.MkdirAll(filepath.Dir(input.OutputPath), 0o755); err != nil {
			return workflow.Retryable("ocr_quality_embed_output_dir_failed", err)
		}
		// #nosec G703 -- output path is explicit operator/workflow input for local artifact export.
		if err := os.WriteFile(input.OutputPath, []byte(embedded), 0o644); err != nil {
			return workflow.Retryable("ocr_quality_embed_write_failed", err)
		}
		outRef, err := step.StoreArtifact("embedded-figures.md", "text/markdown", []byte(embedded), workflow.ArtifactKind("ocr-quality-embedded-markdown"))
		if err != nil {
			return workflow.Retryable("ocr_quality_embed_artifact_failed", err)
		}
		imageIDs := make([]string, 0, len(figures))
		for _, figure := range figures {
			imageBytes, err := os.ReadFile(figure.ImagePath)
			if err != nil {
				return workflow.Retryable("ocr_quality_figure_read_failed", err)
			}
			ref, err := step.StoreArtifact(filepath.Base(figure.ImagePath), "image/png", imageBytes, workflow.ArtifactKind("ocr-quality-extracted-figure"), workflow.ArtifactMetadata(map[string]string{"page": fmt.Sprintf("%03d", figure.PageNumber), "description": figure.Description}))
			if err != nil {
				return workflow.Retryable("ocr_quality_figure_artifact_failed", err)
			}
			imageIDs = append(imageIDs, ref.ID)
		}
		return step.Result(EmbedFiguresResult{InputPath: input.MarkdownPath, OutputPath: input.OutputPath, FiguresDir: input.FiguresDir, FigureCount: len(figures), Figures: figures, OutputRefID: outRef.ID, OutputRefURI: outRef.URI, FigureImageIDs: imageIDs})
	})
}

func ImportLogExecutor() workflow.Executor {
	return workflow.NewTypedExecutor(KindImportLog, func(ctx context.Context, step *workflow.StepContext, input LogImportInput) error {
		result, err := ImportLogFile(input)
		if err != nil {
			return workflow.Permanent("ocr_quality_log_import_failed", err)
		}
		name := strings.TrimSpace(input.ReportName)
		if name == "" {
			name = "log-import.md"
		}
		ref, err := step.StoreArtifact(name, "text/markdown", []byte(result.ReportMarkdown), workflow.ArtifactKind("ocr-quality-log-report"))
		if err != nil {
			return workflow.Retryable("ocr_quality_log_report_artifact_failed", err)
		}
		result.ReportRefID = ref.ID
		result.ReportRefURI = ref.URI
		return step.Result(result)
	})
}

func AssembleReportExecutor() workflow.Executor {
	return workflow.NewTypedExecutor(KindAssembleReport, func(ctx context.Context, step *workflow.StepContext, input ReportInput) error {
		var before QAResult
		_ = step.DependencyData("qa-before", &before)
		var normalized NormalizeResult
		_ = step.DependencyData("normalize-markdown", &normalized)
		var after QAResult
		_ = step.DependencyData("qa-after", &after)
		var b strings.Builder
		b.WriteString("# OCR Quality Pass Report\n\n")
		if input.BookID != "" {
			fmt.Fprintf(&b, "Book ID: `%s`\n\n", input.BookID)
		}
		fmt.Fprintf(&b, "Raw markdown: `%s`\n\n", input.RawMarkdownPath)
		fmt.Fprintf(&b, "Normalized markdown: `%s`\n\n", input.NormalizedPath)
		if input.EmbeddedPath != "" {
			fmt.Fprintf(&b, "Embedded-figure markdown: `%s`\n\n", input.EmbeddedPath)
		}
		fmt.Fprintf(&b, "QA before passed: `%t`\n\n", before.Passed)
		fmt.Fprintf(&b, "QA after passed: `%t`\n\n", after.Passed)
		fmt.Fprintf(&b, "Changed lines: `%d`\n\n", normalized.ChangedLines)
		ref, err := step.StoreArtifact("quality-report.md", "text/markdown", []byte(b.String()), workflow.ArtifactKind("ocr-quality-report"))
		if err != nil {
			return workflow.Retryable("ocr_quality_report_artifact_failed", err)
		}
		return step.Result(ReportResult{ReportRefID: ref.ID, ReportRefURI: ref.URI})
	})
}

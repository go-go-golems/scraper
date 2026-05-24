package ocrmvp

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/go-go-golems/scraper/pkg/workflow"
)

func OCRPageExecutor(projectionName string, client OCRClient) workflow.Executor {
	if client == nil {
		client = DryRunOCRClient{}
	}
	return workflow.NewTypedExecutor(KindOCRPage, func(ctx context.Context, step *workflow.StepContext, input PageOCRInput) error {
		input.PromptVersion = normalizePromptVersion(input.PromptVersion)
		projection, err := step.Projection(projectionNameOrDefault(projectionName))
		if err != nil {
			return workflow.Retryable("ocr_projection_unavailable", err)
		}
		if err := markPageRunning(ctx, projection, input); err != nil {
			return workflow.Retryable("ocr_projection_update_failed", err)
		}
		imageBytes, err := os.ReadFile(input.ImagePath)
		if err != nil {
			markPageError(ctx, projection, input, "ocr_image_read_failed", err)
			return workflow.Permanent("ocr_image_read_failed", err)
		}
		ocrResult, err := client.OCRPage(ctx, input, imageBytes)
		if err != nil {
			markPageError(ctx, projection, input, "ocr_geppetto_failed", err)
			return workflow.Retryable("ocr_geppetto_failed", err)
		}
		text := strings.TrimSpace(ocrResult.Text)
		ref, err := step.StoreArtifact(
			fmt.Sprintf("page_%03d.md", input.PageNumber),
			"text/markdown",
			[]byte(text),
			workflow.ArtifactKind("ocr-markdown"),
			workflow.ArtifactMetadata(pageMetadata(input.BookID, input.PageNumber, input.PromptVersion)),
		)
		if err != nil {
			markPageError(ctx, projection, input, "ocr_artifact_store_failed", err)
			return workflow.Retryable("ocr_artifact_store_failed", err)
		}
		result := PageOCRResult{
			BookID:         input.BookID,
			PageNumber:     input.PageNumber,
			Markdown:       text,
			MarkdownRefID:  ref.ID,
			MarkdownRefURI: ref.URI,
			CharCount:      len(text),
			PromptVersion:  normalizePromptVersion(firstNonEmpty(ocrResult.PromptVersion, input.PromptVersion)),
			Profile:        firstNonEmpty(ocrResult.ProfileSlug, input.Profile),
			Registry:       ocrResult.RegistrySlug,
		}
		if err := markPageDone(ctx, projection, input, result); err != nil {
			return workflow.Retryable("ocr_projection_update_failed", err)
		}
		return step.Result(result)
	})
}

func AssembleMarkdownExecutor(projectionName string) workflow.Executor {
	return workflow.NewTypedExecutor(KindAssembleMarkdown, func(ctx context.Context, step *workflow.StepContext, input AssembleInput) error {
		projection, err := step.Projection(projectionNameOrDefault(projectionName))
		if err != nil {
			return workflow.Retryable("ocr_projection_unavailable", err)
		}
		deps := step.Step().DependsOn
		pages := make([]PageOCRResult, 0, len(deps))
		for _, dep := range deps {
			var page PageOCRResult
			if err := step.DependencyData(dep.OpID, &page); err != nil {
				return workflow.Retryable("ocr_dependency_load_failed", err)
			}
			pages = append(pages, page)
		}
		sort.Slice(pages, func(i, j int) bool { return pages[i].PageNumber < pages[j].PageNumber })
		var out strings.Builder
		for _, page := range pages {
			if out.Len() > 0 {
				out.WriteString("\n\n")
			}
			_, _ = fmt.Fprintf(&out, "<!-- page:%03d -->\n\n", page.PageNumber)
			out.WriteString(strings.TrimSpace(page.Markdown))
			out.WriteString("\n")
		}
		body := out.String()
		ref, err := step.StoreArtifact(
			input.BookID+".md",
			"text/markdown",
			[]byte(body),
			workflow.ArtifactKind("ocr-book-markdown"),
			workflow.ArtifactMetadata(map[string]string{"book_id": input.BookID}),
		)
		if err != nil {
			return workflow.Retryable("ocr_assemble_store_failed", err)
		}
		if err := upsertRun(ctx, projection, input.BookID, "done", len(pages), ref, len(body)); err != nil {
			return workflow.Retryable("ocr_projection_run_update_failed", err)
		}
		return step.Result(AssembleResult{BookID: input.BookID, PageCount: len(pages), MarkdownRefID: ref.ID, MarkdownURI: ref.URI, CharCount: len(body)})
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

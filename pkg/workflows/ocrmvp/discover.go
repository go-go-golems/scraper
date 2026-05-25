package ocrmvp

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/workflow"
)

var pageNumberPattern = regexp.MustCompile(`(\d+)`)

func DiscoverPageImages(input RunInput) ([]PageSpec, error) {
	input = normalizeRunInput(input)
	if err := validateRunInput(input); err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(input.ImageDir, input.PageGlob))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	pages := make([]PageSpec, 0, len(matches))
	for i, path := range matches {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		pageNum := inferPageNumber(path, i+1)
		if input.StartPage > 0 && pageNum < input.StartPage {
			continue
		}
		if input.EndPage > 0 && pageNum > input.EndPage {
			continue
		}
		pages = append(pages, PageSpec{BookID: input.BookID, PageNumber: pageNum, ImagePath: abs})
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no page images matched %s in %s", input.PageGlob, input.ImageDir)
	}
	sort.SliceStable(pages, func(i, j int) bool {
		if pages[i].PageNumber == pages[j].PageNumber {
			return pages[i].ImagePath < pages[j].ImagePath
		}
		return pages[i].PageNumber < pages[j].PageNumber
	})
	return pages, nil
}

func DiscoverPagesExecutor(projectionName string) workflow.Executor {
	return workflow.NewTypedExecutor(KindDiscoverPages, func(ctx context.Context, step *workflow.StepContext, input RunInput) error {
		input = normalizeRunInput(input)
		if err := validateRunInput(input); err != nil {
			return workflow.Permanent("ocr_input_invalid", err)
		}
		projection, err := step.Projection(projectionNameOrDefault(projectionName))
		if err != nil {
			return workflow.Retryable("ocr_projection_unavailable", err)
		}
		if err := EnsureProjectionSchema(ctx, projection); err != nil {
			return workflow.Retryable("ocr_projection_schema_failed", err)
		}
		pages, err := DiscoverPageImages(input)
		if err != nil {
			return workflow.Permanent("ocr_discover_pages_failed", err)
		}
		ocrStepIDs := make([]string, 0, len(pages))
		ocrHandles := make([]workflow.StepHandle, 0, len(pages))
		for _, page := range pages {
			pageInput := PageOCRInput{
				BookID:            input.BookID,
				PageNumber:        page.PageNumber,
				ImagePath:         page.ImagePath,
				Profile:           input.Profile,
				ProfileRegistries: append([]string(nil), input.ProfileRegistries...),
				PromptVersion:     input.PromptVersion,
				ContextBefore:     contextImages(pages, page.PageNumber, input.ContextWindow, -1),
				ContextAfter:      contextImages(pages, page.PageNumber, input.ContextWindow, 1),
				DryRun:            input.DryRun,
			}
			stepID := pageStepID(page.PageNumber)
			if err := upsertPendingPage(ctx, projection, pageInput, string(stepID)); err != nil {
				return workflow.Retryable("ocr_projection_page_seed_failed", err)
			}
			emittedID, err := step.Emit(string(stepID), pageInput, workflow.StepOpts{
				Kind:     KindOCRPage,
				Queue:    QueueOCR,
				Retry:    ocrRetryPolicy(),
				Metadata: pageMetadata(input.BookID, page.PageNumber, input.PromptVersion),
			})
			if err != nil {
				return err
			}
			ocrStepIDs = append(ocrStepIDs, string(emittedID))
			ocrHandles = append(ocrHandles, workflow.StepHandle{ID: emittedID})
		}
		if _, err := step.Emit("assemble-markdown", AssembleInput{BookID: input.BookID}, workflow.StepOpts{
			Kind:      KindAssembleMarkdown,
			Queue:     QueueAssemble,
			DependsOn: workflow.Require(ocrHandles...),
			Metadata:  map[string]string{"book_id": input.BookID},
		}); err != nil {
			return err
		}
		return step.Result(DiscoverResult{BookID: input.BookID, PageCount: len(pages), OCRStepIDs: ocrStepIDs, Pages: pages})
	})
}

func contextImages(pages []PageSpec, pageNum int, window int, direction int) []PageContextImage {
	if window <= 0 || direction == 0 {
		return nil
	}
	byNumber := make(map[int]PageSpec, len(pages))
	for _, page := range pages {
		byNumber[page.PageNumber] = page
	}
	out := make([]PageContextImage, 0, window)
	for offset := 1; offset <= window; offset++ {
		candidate := pageNum + direction*offset
		page, ok := byNumber[candidate]
		if !ok {
			continue
		}
		relation := "next"
		if direction < 0 {
			relation = "previous"
		}
		out = append(out, PageContextImage{PageNumber: page.PageNumber, ImagePath: page.ImagePath, Relation: relation})
	}
	if direction < 0 {
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
	}
	return out
}

func inferPageNumber(path string, fallback int) int {
	base := filepath.Base(path)
	matches := pageNumberPattern.FindAllString(base, -1)
	if len(matches) == 0 {
		return fallback
	}
	pageNum, err := strconv.Atoi(matches[len(matches)-1])
	if err != nil || pageNum <= 0 {
		return fallback
	}
	return pageNum
}

func pageStepID(pageNum int) model.OpID {
	return model.OpID(fmt.Sprintf("ocr-page-%03d", pageNum))
}

func pageMetadata(bookID string, pageNum int, promptVersion string) map[string]string {
	return map[string]string{
		"book_id":        bookID,
		"page":           strconv.Itoa(pageNum),
		"prompt_version": normalizePromptVersion(promptVersion),
	}
}

func projectionNameOrDefault(name string) string {
	if name == "" {
		return ProjectionName
	}
	return name
}

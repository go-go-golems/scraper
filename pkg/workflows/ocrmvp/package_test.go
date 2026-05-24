package ocrmvp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/workflow"
	"github.com/stretchr/testify/require"
)

func TestDiscoverPageImagesOrdersAndFiltersPages(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"page_010.png", "page_002.png", "page_001.png"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644))
	}
	pages, err := DiscoverPageImages(RunInput{BookID: "book", ImageDir: dir, StartPage: 2, EndPage: 10})
	require.NoError(t, err)
	require.Len(t, pages, 2)
	require.Equal(t, 2, pages[0].PageNumber)
	require.Equal(t, 10, pages[1].PageNumber)
}

func TestOCRMVPRunWithFakeClient(t *testing.T) {
	ctx := context.Background()
	workDir := t.TempDir()
	imageDir := filepath.Join(workDir, "pages")
	require.NoError(t, os.MkdirAll(imageDir, 0o755))
	for i := 1; i <= 3; i++ {
		require.NoError(t, os.WriteFile(filepath.Join(imageDir, fmt.Sprintf("page_%03d.png", i)), []byte(fmt.Sprintf("image-%d", i)), 0o644))
	}

	artifactRoot := filepath.Join(workDir, "artifacts")
	rt, err := workflow.NewRuntime(ctx, workflow.Config{
		Store:           workflow.SQLiteStore(filepath.Join(workDir, "engine.db")),
		ArtifactStore:   workflow.NewFileArtifactStore(artifactRoot),
		ProjectionStore: workflow.NewSQLiteProjectionStore(filepath.Join(workDir, "projections")),
		WorkerID:        "ocr-test-worker",
		MaxWorkers:      4,
		PollInterval:    time.Millisecond,
		Queues: map[model.QueueKey]workflow.QueueConfig{
			QueueControl:  {MaxWorkers: 1},
			QueueOCR:      {MaxWorkers: 3},
			QueueAssemble: {MaxWorkers: 1},
		},
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, rt.Close()) }()

	fakeClient := OCRClientFunc(func(ctx context.Context, input PageOCRInput, imageBytes []byte) (OCRTextResult, error) {
		return OCRTextResult{
			Text:          fmt.Sprintf("## Page %03d\n\nFake OCR for %s using %s.", input.PageNumber, string(imageBytes), RenderPagePrompt(input)[:20]),
			ProfileSlug:   "fake-profile",
			RegistrySlug:  "fake-registry",
			PromptVersion: input.PromptVersion,
		}, nil
	})
	require.NoError(t, Register(rt, Config{Client: fakeClient}))

	run, err := rt.StartRun(ctx, PackageName, RunInput{
		BookID:        "book-a",
		ImageDir:      imageDir,
		PromptVersion: "test-prompt-v1",
	}, workflow.WithRunID("ocr-run-1"))
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		_, err := rt.RunOnce(ctx)
		require.NoError(t, err)
		wf, err := rt.Workflow(ctx, run.ID)
		require.NoError(t, err)
		if wf.Status == model.WorkflowStatusSucceeded {
			break
		}
	}
	wf, err := rt.Workflow(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusSucceeded, wf.Status)

	discover, err := rt.Result(ctx, run.ID, "discover-pages")
	require.NoError(t, err)
	require.JSONEq(t, `{"book_id":"book-a","page_count":3,"ocr_step_ids":["ocr-page-001","ocr-page-002","ocr-page-003"],"pages":[{"book_id":"book-a","page_number":1,"image_path":"`+filepath.Join(imageDir, "page_001.png")+`"},{"book_id":"book-a","page_number":2,"image_path":"`+filepath.Join(imageDir, "page_002.png")+`"},{"book_id":"book-a","page_number":3,"image_path":"`+filepath.Join(imageDir, "page_003.png")+`"}]}`, string(discover.Data))

	pageResult, err := rt.Result(ctx, run.ID, "ocr-page-002")
	require.NoError(t, err)
	require.Len(t, pageResult.Artifacts, 1)
	require.Equal(t, "external-artifact-ref", pageResult.Artifacts[0].Kind)

	assembleResult, err := rt.Result(ctx, run.ID, "assemble-markdown")
	require.NoError(t, err)
	require.Len(t, assembleResult.Artifacts, 1)
	reader, _, err := workflow.NewFileArtifactStore(artifactRoot).Open(ctx, string(assembleResult.Artifacts[0].ID))
	require.NoError(t, err)
	defer reader.Close()
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Contains(t, string(body), "<!-- page:001 -->")
	require.Contains(t, string(body), "<!-- page:002 -->")
	require.Contains(t, string(body), "<!-- page:003 -->")
	require.Contains(t, string(body), "Fake OCR for image-2")

	projection, err := rt.Projection(ctx, ProjectionName)
	require.NoError(t, err)
	rows, err := projection.Query(ctx, `SELECT page_num, status, profile, registry FROM pages WHERE book_id = ? ORDER BY page_num`, "book-a")
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, int64(1), rows[0]["page_num"])
	require.Equal(t, "done", rows[0]["status"])
	require.Equal(t, "fake-profile", rows[0]["profile"])
	require.Equal(t, "fake-registry", rows[0]["registry"])

	runRows, err := projection.Query(ctx, `SELECT status, page_count FROM runs WHERE book_id = ?`, "book-a")
	require.NoError(t, err)
	require.Len(t, runRows, 1)
	require.Equal(t, "done", runRows[0]["status"])
	require.Equal(t, int64(3), runRows[0]["page_count"])
}

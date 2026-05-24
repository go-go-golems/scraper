package ocrmvp

import (
	"context"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflow"
)

func EnsureProjectionSchema(ctx context.Context, projection workflow.Projection) error {
	_, err := projection.Exec(ctx, `CREATE TABLE IF NOT EXISTS pages (
		book_id TEXT NOT NULL,
		page_num INTEGER NOT NULL,
		image_path TEXT NOT NULL,
		status TEXT NOT NULL,
		ocr_step_id TEXT,
		markdown_artifact_id TEXT,
		markdown_artifact_uri TEXT,
		char_count INTEGER NOT NULL DEFAULT 0,
		error_code TEXT,
		error_message TEXT,
		prompt_version TEXT,
		profile TEXT,
		registry TEXT,
		updated_at TEXT NOT NULL,
		PRIMARY KEY(book_id, page_num)
	)`)
	if err != nil {
		return err
	}
	_, err = projection.Exec(ctx, `CREATE TABLE IF NOT EXISTS runs (
		book_id TEXT PRIMARY KEY,
		status TEXT NOT NULL,
		page_count INTEGER NOT NULL DEFAULT 0,
		markdown_artifact_id TEXT,
		markdown_artifact_uri TEXT,
		char_count INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL
	)`)
	return err
}

func upsertPendingPage(ctx context.Context, projection workflow.Projection, input PageOCRInput, stepID string) error {
	_, err := projection.Exec(ctx, `INSERT INTO pages (
		book_id, page_num, image_path, status, ocr_step_id, prompt_version, profile, updated_at
	) VALUES (?, ?, ?, 'pending', ?, ?, ?, ?)
	ON CONFLICT(book_id, page_num) DO UPDATE SET
		image_path = excluded.image_path,
		status = 'pending',
		ocr_step_id = excluded.ocr_step_id,
		markdown_artifact_id = NULL,
		markdown_artifact_uri = NULL,
		char_count = 0,
		error_code = NULL,
		error_message = NULL,
		prompt_version = excluded.prompt_version,
		profile = excluded.profile,
		registry = NULL,
		updated_at = excluded.updated_at`,
		input.BookID, input.PageNumber, input.ImagePath, stepID, normalizePromptVersion(input.PromptVersion), input.Profile, nowText())
	return err
}

func markPageRunning(ctx context.Context, projection workflow.Projection, input PageOCRInput) error {
	_, err := projection.Exec(ctx, `UPDATE pages SET status = 'running', error_code = NULL, error_message = NULL, updated_at = ? WHERE book_id = ? AND page_num = ?`, nowText(), input.BookID, input.PageNumber)
	return err
}

func markPageDone(ctx context.Context, projection workflow.Projection, input PageOCRInput, result PageOCRResult) error {
	_, err := projection.Exec(ctx, `UPDATE pages SET
		status = 'done',
		markdown_artifact_id = ?,
		markdown_artifact_uri = ?,
		char_count = ?,
		error_code = NULL,
		error_message = NULL,
		prompt_version = ?,
		profile = ?,
		registry = ?,
		updated_at = ?
	WHERE book_id = ? AND page_num = ?`,
		result.MarkdownRefID, result.MarkdownRefURI, result.CharCount, result.PromptVersion, result.Profile, result.Registry, nowText(), input.BookID, input.PageNumber)
	return err
}

func markPageError(ctx context.Context, projection workflow.Projection, input PageOCRInput, code string, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	_, _ = projection.Exec(ctx, `UPDATE pages SET status = 'error', error_code = ?, error_message = ?, updated_at = ? WHERE book_id = ? AND page_num = ?`, code, msg, nowText(), input.BookID, input.PageNumber)
}

func upsertRun(ctx context.Context, projection workflow.Projection, bookID, status string, pageCount int, ref workflow.ArtifactRef, charCount int) error {
	_, err := projection.Exec(ctx, `INSERT INTO runs (
		book_id, status, page_count, markdown_artifact_id, markdown_artifact_uri, char_count, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(book_id) DO UPDATE SET
		status = excluded.status,
		page_count = excluded.page_count,
		markdown_artifact_id = excluded.markdown_artifact_id,
		markdown_artifact_uri = excluded.markdown_artifact_uri,
		char_count = excluded.char_count,
		updated_at = excluded.updated_at`,
		bookID, status, pageCount, ref.ID, ref.URI, charCount, nowText())
	return err
}

func nowText() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

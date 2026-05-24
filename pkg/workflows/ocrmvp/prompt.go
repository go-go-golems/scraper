package ocrmvp

import "fmt"

const DefaultPromptVersion = "ocr-mvp-universal-v1"

const OCRSystemPrompt = `You are a precise OCR transcription engine. Transcribe only visible page content into clean markdown.`

func RenderPagePrompt(input PageOCRInput) string {
	version := input.PromptVersion
	if version == "" {
		version = DefaultPromptVersion
	}
	return fmt.Sprintf(`Transcribe this scanned book/report page into clean markdown.

Rules:
1. Output only markdown. No commentary.
2. Preserve headings, paragraphs, footnotes, citations, math, code, and tables.
3. If the page is blank, output an empty string.
4. If an image/figure/diagram appears, insert exactly one single-line marker:
   [IMAGE: concise description of what the figure shows]
5. Do not include standalone page numbers.
6. Do not duplicate text.
7. Do not add text that is not visible on the page.

Book ID: %s
Page number: %03d
Prompt version: %s
`, input.BookID, input.PageNumber, version)
}

func normalizePromptVersion(version string) string {
	if version == "" {
		return DefaultPromptVersion
	}
	return version
}

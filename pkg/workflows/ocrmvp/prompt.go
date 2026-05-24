package ocrmvp

import "fmt"

const DefaultPromptVersion = "ocr-mvp-universal-v1"

const PromptVersionQualityV2 = "ocr-quality-v2"

const OCRSystemPrompt = `You are a precise OCR transcription engine. Transcribe only visible page content into clean markdown.`

func RenderPagePrompt(input PageOCRInput) string {
	version := normalizePromptVersion(input.PromptVersion)
	switch version {
	case PromptVersionQualityV2:
		return renderQualityV2Prompt(input, version)
	default:
		return renderUniversalV1Prompt(input, version)
	}
}

func renderUniversalV1Prompt(input PageOCRInput, version string) string {
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

func renderQualityV2Prompt(input PageOCRInput, version string) string {
	return fmt.Sprintf(`Transcribe this scanned technical-report page into faithful, clean markdown.

Output contract:
1. Output only the transcription markdown. Do not explain your work.
2. Preserve only visible page content. Do not invent connecting text.
3. Do not include standalone running page numbers, footer folios, or scanner artifacts.
4. Preserve original spelling unless it is clearly OCR noise. Do not modernize terminology such as "data base".

Page-type rules:
- Title page: transcribe the visible title, report number, author, institution, date, and copyright as text. Do not replace the title page with an [IMAGE: ...] marker.
- Blank or intentionally blank page: output exactly [BLANK PAGE].
- Table of contents or table of figures: preserve the list style consistently. Do not use markdown bullets. Keep chapter/section/figure labels, punctuation, dot leaders when visible, and final page numbers. Continuation pages must keep the same style as the previous list page; do not suddenly switch formats.
- Body text: preserve headings and paragraphs. If a paragraph begins or ends mid-sentence because of a page boundary, transcribe exactly the visible fragment without adding ellipses or explanatory notes.
- Figures/diagrams: transcribe any visible caption as text. Then add exactly one marker on the next line: [FIGURE: concise description]. Do not use this marker for title pages or ordinary decorated text.
- Tables: preserve rows and columns as markdown tables when readable; otherwise use aligned plain text.

Markdown style:
- Use # for visible chapter titles and ##/### for visible section headings when the page clearly begins a heading.
- Do not promote every figure caption to a markdown heading.
- Keep emphasis only when visible or semantically necessary for terms already emphasized on the page.
- Preserve citations and bracketed references exactly as visible.

Quality checklist before final answer:
- No duplicated lines.
- No omitted visible headings.
- No page footer number included.
- List pages use one consistent style across the whole page.
- Title pages are text, not image descriptions.

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

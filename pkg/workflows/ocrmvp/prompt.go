package ocrmvp

import "fmt"

const DefaultPromptVersion = "ocr-mvp-universal-v1"

const PromptVersionQualityV2 = "ocr-quality-v2"

const PromptVersionQualityV3ListDiplomatic = "ocr-quality-v3-list-diplomatic"

const OCRSystemPrompt = `You are a precise OCR transcription engine. Transcribe only visible page content into clean markdown.`

func RenderPagePrompt(input PageOCRInput) string {
	version := normalizePromptVersion(input.PromptVersion)
	switch version {
	case PromptVersionQualityV3ListDiplomatic:
		return renderQualityV3ListDiplomaticPrompt(input, version)
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

func renderQualityV3ListDiplomaticPrompt(input PageOCRInput, version string) string {
	return fmt.Sprintf(`Transcribe this scanned technical-report page into faithful, clean markdown.

Output contract:
1. Output only the transcription markdown. Do not explain your work.
2. Transcribe only visible page content. Do not add summaries, comments, or inferred missing text.
3. Exclude standalone running page numbers, footer folios, scanner borders, and scanner artifacts.
4. Preserve original spelling and historical terminology, including "data base" when visible.
5. Preserve visible text order from top to bottom.

Global normalization policy:
- Prefer readable markdown over visual line wrapping for normal prose.
- Join wrapped title lines when they are clearly one title phrase, unless the line break is semantically meaningful.
- Do not duplicate a line just because it appears as both a visual heading and a list row. If the same visible line is repeated on the page, transcribe it once per visible occurrence; otherwise do not invent duplicates.

Page-type rules:
- Blank or intentionally blank page: output exactly [BLANK PAGE].
- Title/front-matter page: transcribe visible report number, title, author, institution, date, copyright, and notes as text. Do not use an image marker for title pages. Normalize the main title to one readable line when it is a single phrase.
- Abstract/acknowledgments/body page: preserve headings and paragraphs. If a paragraph begins or ends mid-sentence because of the page boundary, transcribe exactly the visible fragment without ellipses.
- Table of Contents pages: use a diplomatic plain-text list, not markdown bullets and not markdown headings. Preserve chapter titles, section numbers, section titles, dot leaders or spacing when visible, and final page numbers. Continuation pages must use the same plain-text style as the first Table of Contents page. Never duplicate a chapter title line.
- Table of Figures pages: use a diplomatic plain-text list, not markdown bullets and not markdown headings. Preserve each entry as "Figure N-M: Title ... page" or the closest visible punctuation. Preserve dot leaders or spacing when visible and final page numbers. Continuation pages must use the same style as the first Table of Figures page.
- Figures/diagrams outside list pages: transcribe any visible caption as text, then add exactly one marker on the next line: [FIGURE: concise description]. Do not use this marker for title pages, contents pages, or table-of-figures pages.
- Tables: preserve rows and columns as markdown tables when readable; otherwise use aligned plain text.

List-page checklist:
- If this page is a Table of Contents or Table of Figures page, do not use #, ##, ###, -, *, or numbered markdown-list syntax for the list.
- Keep one entry per visible row.
- Keep page numbers at the end of entries.
- Exclude the page's own footer number.
- If a page continues a list without a repeated heading, do not invent a heading. If the heading is visibly repeated, transcribe it once.

Quality checklist before final answer:
- No duplicated lines.
- No invented continuation notes.
- No footer page number included.
- Title page text is readable and not split by decorative visual wrapping.
- Contents/list pages are plain text and internally consistent.

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

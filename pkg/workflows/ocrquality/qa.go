package ocrquality

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	figureMarkerPattern = regexp.MustCompile(`(?m)^\[FIGURE:`)
	bulletLinePattern   = regexp.MustCompile(`^\s*[-*]\s+`)
	headingLinePattern  = regexp.MustCompile(`^\s*#{1,6}\s+`)
)

func AnalyzeMarkdown(markdown string, input QAInput) QAResult {
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

	pages := SplitPages(markdown)
	observed := PageNumbers(pages)
	result := QAResult{
		MarkdownPath:       input.MarkdownPath,
		ExpectedPages:      input.ExpectedPages,
		PageMarkersFound:   len(pages),
		ObservedPages:      observed,
		FigureMarkers:      len(figureMarkerPattern.FindAllStringIndex(markdown, -1)),
		KnownBadTermHits:   map[string]int{},
		ExpectedStringHits: map[string]bool{},
	}

	if input.ExpectedPages > 0 && !sameInts(observed, expectedPageNumbers(input.ExpectedPages)) {
		result.Findings = append(result.Findings, QAFinding{
			Severity: SeverityError,
			Code:     "page_continuity",
			Message:  fmt.Sprintf("observed pages %v do not match expected 1..%d", observed, input.ExpectedPages),
		})
	}

	for _, term := range input.KnownBadTerms {
		count := strings.Count(markdown, term)
		if count == 0 {
			continue
		}
		result.KnownBadTermHits[term] = count
		result.Findings = append(result.Findings, QAFinding{
			Severity: SeverityError,
			Code:     "known_bad_term",
			Message:  fmt.Sprintf("known bad term %q found %d time(s)", term, count),
			Line:     lineNumber(markdown, term),
		})
	}
	if len(result.KnownBadTermHits) == 0 {
		result.KnownBadTermHits = nil
	}

	for _, expected := range input.ExpectedStrings {
		hit := strings.Contains(markdown, expected)
		result.ExpectedStringHits[expected] = hit
		if !hit {
			result.Findings = append(result.Findings, QAFinding{
				Severity: SeverityError,
				Code:     "expected_string_missing",
				Message:  fmt.Sprintf("expected string %q is missing", expected),
			})
		}
	}

	for _, page := range pages {
		for _, dup := range adjacentDuplicateLines(page) {
			result.DuplicateLines = append(result.DuplicateLines, dup)
			result.Findings = append(result.Findings, dup)
		}
	}

	listPages := intSet(input.ListPages)
	for _, page := range pages {
		if !listPages[page.Number] {
			continue
		}
		check := PageStyleCheck{Page: page.Number, Label: listPageLabel(page.Number)}
		for _, line := range strings.Split(page.Text, "\n") {
			if bulletLinePattern.MatchString(line) {
				check.MarkdownBulletLines++
				if len(check.BulletSamples) < 3 {
					check.BulletSamples = append(check.BulletSamples, strings.TrimSpace(line))
				}
			}
			if headingLinePattern.MatchString(line) {
				check.MarkdownHeadingLines++
				if len(check.HeadingSamples) < 3 {
					check.HeadingSamples = append(check.HeadingSamples, strings.TrimSpace(line))
				}
			}
		}
		if check.MarkdownBulletLines > 0 {
			result.Findings = append(result.Findings, QAFinding{
				Severity: SeverityWarning,
				Code:     "list_page_markdown_bullets",
				Page:     page.Number,
				Message:  fmt.Sprintf("list page has %d markdown bullet line(s)", check.MarkdownBulletLines),
			})
		}
		if check.MarkdownHeadingLines > 0 {
			result.Findings = append(result.Findings, QAFinding{
				Severity: SeverityWarning,
				Code:     "list_page_markdown_headings",
				Page:     page.Number,
				Message:  fmt.Sprintf("list page has %d markdown heading line(s)", check.MarkdownHeadingLines),
			})
		}
		result.ListPageChecks = append(result.ListPageChecks, check)
	}
	sort.Slice(result.ListPageChecks, func(i, j int) bool { return result.ListPageChecks[i].Page < result.ListPageChecks[j].Page })

	result.Passed = true
	for _, finding := range result.Findings {
		if finding.Severity == SeverityError || finding.Severity == SeverityWarning {
			result.Passed = false
			break
		}
	}
	result.ReportMarkdown = RenderQAReport(result)
	return result
}

func adjacentDuplicateLines(page MarkdownPage) []QAFinding {
	var findings []QAFinding
	var prev string
	for i, raw := range strings.Split(page.Text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			prev = ""
			continue
		}
		if line == prev {
			findings = append(findings, QAFinding{
				Severity: SeverityError,
				Code:     "adjacent_duplicate_line",
				Page:     page.Number,
				Line:     lineNumberInPage(page, i+1),
				Message:  fmt.Sprintf("adjacent duplicate non-empty line: %s", line),
			})
		}
		prev = line
	}
	return findings
}

func RenderQAReport(result QAResult) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("docType: reference\nstatus: active\nintent: short-term\ntopics:\n  - ocr\n  - experiments\ncreated: 2026-05-24\nupdated: 2026-05-24\n---\n\n")
	b.WriteString("# OCR Markdown QA Report\n\n")
	if result.MarkdownPath != "" {
		fmt.Fprintf(&b, "Input: `%s`\n\n", result.MarkdownPath)
	}
	b.WriteString("## Summary\n\n")
	fmt.Fprintf(&b, "- Page markers found: %d\n", result.PageMarkersFound)
	fmt.Fprintf(&b, "- Expected page markers: %d\n", result.ExpectedPages)
	fmt.Fprintf(&b, "- Figure markers: %d\n\n", result.FigureMarkers)

	b.WriteString("## Known bad term checks\n\n")
	if len(result.KnownBadTermHits) == 0 {
		b.WriteString("- PASS: no known bad terms found.\n\n")
	} else {
		for _, term := range sortedKeys(result.KnownBadTermHits) {
			fmt.Fprintf(&b, "- `%s`: %d hit(s)\n", term, result.KnownBadTermHits[term])
		}
		b.WriteString("\n")
	}

	b.WriteString("## Expected string checks\n\n")
	keys := make([]string, 0, len(result.ExpectedStringHits))
	for key := range result.ExpectedStringHits {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if result.ExpectedStringHits[key] {
			fmt.Fprintf(&b, "- PASS: `%s`\n", key)
		} else {
			fmt.Fprintf(&b, "- MISSING: `%s`\n", key)
		}
	}
	b.WriteString("\n")

	b.WriteString("## Adjacent duplicate non-empty lines\n\n")
	if len(result.DuplicateLines) == 0 {
		b.WriteString("- PASS: no adjacent duplicate non-empty lines found.\n\n")
	} else {
		for _, finding := range result.DuplicateLines {
			fmt.Fprintf(&b, "- page %03d, local line %d: %s\n", finding.Page, finding.Line, finding.Message)
		}
		b.WriteString("\n")
	}

	b.WriteString("## List-page style checks\n\n")
	for _, check := range result.ListPageChecks {
		fmt.Fprintf(&b, "### Page %03d: %s\n", check.Page, check.Label)
		fmt.Fprintf(&b, "- Markdown bullet lines: %d\n", check.MarkdownBulletLines)
		fmt.Fprintf(&b, "- Markdown heading lines: %d\n\n", check.MarkdownHeadingLines)
	}

	b.WriteString("## Verdict\n\n")
	if result.Passed {
		b.WriteString("PASS for automated checks. Manual visual spot-checking is still required for OCR accuracy.\n")
	} else {
		b.WriteString("REVIEW REQUIRED: one or more automated checks produced findings.\n")
	}
	return b.String()
}

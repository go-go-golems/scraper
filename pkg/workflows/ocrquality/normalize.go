package ocrquality

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	leaderLinePattern = regexp.MustCompile(`^(\S.*?)(?:\s*\.\s*){3,}\s*(\d{1,3})\s*$`)
	spacedPagePattern = regexp.MustCompile(`^(\S.*?\S)\s{2,}(\d{1,3})\s*$`)
)

type NormalizeOptions struct {
	ListPages []int
}

type NormalizeStats struct {
	Changed      bool
	ChangedLines int
	ChangedPages []int
}

func NormalizeMarkdown(markdown string, opts NormalizeOptions) (string, NormalizeStats) {
	if len(opts.ListPages) == 0 {
		opts.ListPages = defaultListPages()
	}
	listPages := intSet(opts.ListPages)
	matches := pageMarkerPattern.FindAllStringSubmatchIndex(markdown, -1)
	if len(matches) == 0 {
		return markdown, NormalizeStats{}
	}
	var out strings.Builder
	out.WriteString(strings.TrimRight(markdown[:matches[0][0]], "\n"))
	out.WriteString("\n")
	stats := NormalizeStats{}
	changedPageSet := map[int]bool{}
	for i, match := range matches {
		pageNum := 0
		_, _ = fmt.Sscanf(markdown[match[2]:match[3]], "%d", &pageNum)
		start := match[1]
		end := len(markdown)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		body := strings.Trim(markdown[start:end], "\n")
		if listPages[pageNum] {
			var pageChanges int
			body, pageChanges = normalizeListPageBody(body)
			if pageChanges > 0 {
				stats.Changed = true
				stats.ChangedLines += pageChanges
				changedPageSet[pageNum] = true
			}
		}
		fmt.Fprintf(&out, "<!-- page:%03d -->\n\n", pageNum)
		out.WriteString(body)
		out.WriteString("\n\n")
	}
	for page := range changedPageSet {
		stats.ChangedPages = append(stats.ChangedPages, page)
	}
	for i := 0; i < len(stats.ChangedPages); i++ {
		for j := i + 1; j < len(stats.ChangedPages); j++ {
			if stats.ChangedPages[j] < stats.ChangedPages[i] {
				stats.ChangedPages[i], stats.ChangedPages[j] = stats.ChangedPages[j], stats.ChangedPages[i]
			}
		}
	}
	return strings.TrimRight(out.String(), "\n") + "\n", stats
}

func normalizeListPageBody(body string) (string, int) {
	lines := strings.Split(body, "\n")
	changes := 0
	out := make([]string, 0, len(lines))
	blankCount := 0
	for _, line := range lines {
		normalized := normalizeListLine(line)
		if normalized != strings.TrimRight(line, " \t") {
			changes++
		}
		if strings.TrimSpace(normalized) == "" {
			blankCount++
			if blankCount <= 2 {
				out = append(out, "")
			}
			continue
		}
		blankCount = 0
		out = append(out, normalized)
	}
	return strings.Trim(strings.Join(out, "\n"), "\n"), changes
}

func normalizeListLine(line string) string {
	stripped := strings.TrimRight(line, " \t")
	if strings.TrimSpace(stripped) == "" {
		return ""
	}
	for _, pattern := range []*regexp.Regexp{leaderLinePattern, spacedPagePattern} {
		match := pattern.FindStringSubmatch(stripped)
		if len(match) != 3 {
			continue
		}
		label := strings.Join(strings.Fields(match[1]), " ")
		if len(label) < 3 {
			return stripped
		}
		return fmt.Sprintf("%s ... %s", label, match[2])
	}
	return stripped
}

func UnifiedLineDiff(oldName, newName, oldText, newText string) string {
	if oldText == newText {
		return ""
	}
	oldLines := strings.SplitAfter(oldText, "\n")
	newLines := strings.SplitAfter(newText, "\n")
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n", oldName)
	fmt.Fprintf(&b, "+++ %s\n", newName)
	fmt.Fprintf(&b, "@@ -1,%d +1,%d @@\n", len(oldLines), len(newLines))
	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}
	for i := 0; i < maxLines; i++ {
		var oldLine, newLine string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}
		switch {
		case oldLine == newLine:
			b.WriteString(" " + ensureLine(oldLine))
		case oldLine == "":
			b.WriteString("+" + ensureLine(newLine))
		case newLine == "":
			b.WriteString("-" + ensureLine(oldLine))
		default:
			b.WriteString("-" + ensureLine(oldLine))
			b.WriteString("+" + ensureLine(newLine))
		}
	}
	return b.String()
}

func ensureLine(s string) string {
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

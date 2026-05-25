package ocrquality

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var pageMarkerPattern = regexp.MustCompile(`(?m)^<!-- page:(\d{3}) -->\s*$`)

type MarkdownPage struct {
	Number int
	Text   string
}

func SplitPages(markdown string) []MarkdownPage {
	matches := pageMarkerPattern.FindAllStringSubmatchIndex(markdown, -1)
	pages := make([]MarkdownPage, 0, len(matches))
	for i, match := range matches {
		pageNum, err := strconv.Atoi(markdown[match[2]:match[3]])
		if err != nil {
			continue
		}
		start := match[1]
		end := len(markdown)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		pages = append(pages, MarkdownPage{Number: pageNum, Text: strings.Trim(markdown[start:end], "\n")})
	}
	return pages
}

func PageNumbers(pages []MarkdownPage) []int {
	out := make([]int, len(pages))
	for i, page := range pages {
		out[i] = page.Number
	}
	return out
}

func expectedPageNumbers(n int) []int {
	if n <= 0 {
		return nil
	}
	out := make([]int, n)
	for i := range out {
		out[i] = i + 1
	}
	return out
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func lineNumber(text, needle string) int {
	idx := strings.Index(text, needle)
	if idx < 0 {
		return 0
	}
	return strings.Count(text[:idx], "\n") + 1
}

func lineNumberInPage(page MarkdownPage, localLine int) int {
	if localLine <= 0 {
		return 0
	}
	return localLine
}

func defaultKnownBadTerms() []string {
	return []string{"DiRed", "Streamer", "PPSBase", "Ciccarrelli", "[IMAGE:"}
}

func defaultExpectedStrings() []string {
	return []string{
		"Presentation Based User Interfaces",
		"This blank page was inserted to preserve pagination.",
		"Figure 4-1: Dired Model",
		"Figure 4-9: Sample Steamer Schematic",
		"Figure 5-1: PSBase Support of PPS Components",
		"Chapter Two",
		"The Primitive Presentation System (PPS) Model",
		"2.1 PPSCalc",
	}
}

func defaultListPages() []int { return []int{6, 7, 8, 9} }

func listPageLabel(page int) string {
	switch page {
	case 6:
		return "Table of Contents"
	case 7:
		return "Table of Contents continuation"
	case 8:
		return "Table of Figures"
	case 9:
		return "Table of Figures continuation"
	default:
		return fmt.Sprintf("List page %03d", page)
	}
}

func intSet(values []int) map[int]bool {
	m := make(map[int]bool, len(values))
	for _, value := range values {
		m[value] = true
	}
	return m
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

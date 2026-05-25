package ocrquality

import (
	"strings"
	"testing"
)

const sampleMarkdown = `<!-- page:001 -->

Technical Report 794

Presentation Based User Interfaces

<!-- page:002 -->

This blank page was inserted to preserve pagination.

<!-- page:003 -->

Figure 4-1: Dired Model ... 72
Figure 4-9: Sample Steamer Schematic ... 91
Figure 5-1: PSBase Support of PPS Components ... 101
Chapter Two
The Primitive Presentation System (PPS) Model
2.1 PPSCalc
`

func TestAnalyzeMarkdownPassesKnownFixture(t *testing.T) {
	result := AnalyzeMarkdown(sampleMarkdown, QAInput{ExpectedPages: 3, ListPages: []int{3}})
	if !result.Passed {
		t.Fatalf("expected QA to pass, findings=%+v", result.Findings)
	}
	if result.PageMarkersFound != 3 {
		t.Fatalf("expected 3 page markers, got %d", result.PageMarkersFound)
	}
}

func TestAnalyzeMarkdownFindsKnownBadTerm(t *testing.T) {
	result := AnalyzeMarkdown(strings.Replace(sampleMarkdown, "Dired", "DiRed", 1), QAInput{ExpectedPages: 3, ListPages: []int{3}})
	if result.Passed {
		t.Fatalf("expected QA to fail")
	}
	if result.KnownBadTermHits["DiRed"] != 1 {
		t.Fatalf("expected DiRed hit, got %#v", result.KnownBadTermHits)
	}
}

func TestAnalyzeMarkdownFindsAdjacentDuplicate(t *testing.T) {
	md := sampleMarkdown + "duplicate\nduplicate\n"
	result := AnalyzeMarkdown(md, QAInput{ExpectedPages: 3, ListPages: []int{3}})
	if result.Passed {
		t.Fatalf("expected QA to fail")
	}
	if len(result.DuplicateLines) != 1 {
		t.Fatalf("expected one duplicate finding, got %d", len(result.DuplicateLines))
	}
}

func TestNormalizeMarkdownListPages(t *testing.T) {
	input := `<!-- page:006 -->

Table of Contents

Chapter One: Introduction and Overview..........................................................8
1.1 The Primitive Presentation System Model                            9

<!-- page:010 -->

Not a list..........................................................8
`
	out, stats := NormalizeMarkdown(input, NormalizeOptions{ListPages: []int{6}})
	if !stats.Changed || stats.ChangedLines != 2 {
		t.Fatalf("expected two changed lines, got %+v", stats)
	}
	if !strings.Contains(out, "Chapter One: Introduction and Overview ... 8") {
		t.Fatalf("expected normalized chapter line:\n%s", out)
	}
	if !strings.Contains(out, "1.1 The Primitive Presentation System Model ... 9") {
		t.Fatalf("expected normalized section line:\n%s", out)
	}
	if !strings.Contains(out, "Not a list..........................................................8") {
		t.Fatalf("non-list page should remain unchanged:\n%s", out)
	}
}

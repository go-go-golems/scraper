package ocrmvp

import "testing"

func TestContextImagesAroundPage(t *testing.T) {
	pages := []PageSpec{
		{PageNumber: 1, ImagePath: "page_001.png"},
		{PageNumber: 2, ImagePath: "page_002.png"},
		{PageNumber: 3, ImagePath: "page_003.png"},
		{PageNumber: 4, ImagePath: "page_004.png"},
	}
	before := contextImages(pages, 3, 2, -1)
	if len(before) != 2 || before[0].PageNumber != 1 || before[1].PageNumber != 2 {
		t.Fatalf("unexpected before context: %#v", before)
	}
	after := contextImages(pages, 3, 2, 1)
	if len(after) != 1 || after[0].PageNumber != 4 {
		t.Fatalf("unexpected after context: %#v", after)
	}
}

func TestNormalizeRunInputCapsContextWindow(t *testing.T) {
	input := normalizeRunInput(RunInput{BookID: "b", ImageDir: "/tmp", ContextWindow: 99})
	if input.ContextWindow != 2 {
		t.Fatalf("expected context window cap 2, got %d", input.ContextWindow)
	}
	input = normalizeRunInput(RunInput{BookID: "b", ImageDir: "/tmp", ContextWindow: -1})
	if input.ContextWindow != 0 {
		t.Fatalf("expected negative context window to become 0, got %d", input.ContextWindow)
	}
}

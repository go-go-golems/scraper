package ocrquality

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbedExtractedFigures(t *testing.T) {
	dir := t.TempDir()
	imageDir := filepath.Join(dir, "pages")
	figDir := filepath.Join(dir, "figures")
	writeTestPageImage(t, imageDir, 13)

	md := "<!-- page:013 -->\n\nFigure 1-1: Test\n\n[FIGURE: black rectangle]\n"
	out, figures, err := EmbedExtractedFigures(md, FigureExtractionOptions{ImageDir: imageDir, OutputDir: figDir})
	if err != nil {
		t.Fatal(err)
	}
	if len(figures) != 1 {
		t.Fatalf("expected one figure, got %d", len(figures))
	}
	if !strings.Contains(out, "![black rectangle](figures/page_013_figure_01.png)") {
		t.Fatalf("expected markdown image link, got:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(figDir, "page_013_figure_01.png")); err != nil {
		t.Fatal(err)
	}
}

func TestEmbedExtractedFiguresSynthesizesDiagramPageMarkers(t *testing.T) {
	dir := t.TempDir()
	imageDir := filepath.Join(dir, "pages")
	figDir := filepath.Join(dir, "figures")
	writeTestPageImage(t, imageDir, 15)

	md := `<!-- page:015 -->

Figure 1-2: The Representation Shift Model

User -> Presentation Editor
editing commands

Presentation Editor -> Presentation Data Base

Presenter <- query (GET-DB) - - - -> Application Data Base
`
	out, figures, err := EmbedExtractedFigures(md, FigureExtractionOptions{ImageDir: imageDir, OutputDir: figDir})
	if err != nil {
		t.Fatal(err)
	}
	if len(figures) != 1 {
		t.Fatalf("expected synthesized figure, got %d", len(figures))
	}
	if figures[0].PageNumber != 15 || !strings.Contains(figures[0].Description, "Representation Shift Model") {
		t.Fatalf("unexpected figure metadata: %+v", figures[0])
	}
	if !strings.Contains(out, "![Full-page diagram showing The Representation Shift Model](figures/page_015_figure_01.png)") {
		t.Fatalf("expected synthesized markdown image link, got:\n%s", out)
	}
}

func TestEmbedExtractedFiguresDoesNotSynthesizeTableOfFiguresRows(t *testing.T) {
	md := `<!-- page:008 -->

Figure 1-2: The Representation Shift Model ... 13
Figure 1-3: The Primitive Presentation System (PPS) Model ... 15
Figure 1-4: Structure of PSBase ... 19
`
	out := synthesizeMissingFigureMarkers(md)
	if strings.Contains(out, "[FIGURE:") {
		t.Fatalf("did not expect synthesized marker for table-of-figures rows:\n%s", out)
	}
}

func writeTestPageImage(t *testing.T, imageDir string, page int) {
	t.Helper()
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.White)
		}
	}
	for y := 30; y < 70; y++ {
		for x := 25; x < 75; x++ {
			img.Set(x, y, color.Black)
		}
	}
	f, err := os.Create(filepath.Join(imageDir, fmt.Sprintf("page_%03d.png", page)))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

package ocrquality

import (
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
	f, err := os.Create(filepath.Join(imageDir, "page_013.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

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

package ocrquality

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var figureMarkerLinePattern = regexp.MustCompile(`^\[FIGURE:\s*(.*?)\]\s*$`)
var figureCaptionLinePattern = regexp.MustCompile(`^Figure\s+\d+-\d+:\s+(.+?)\s*$`)

type FigureExtractionOptions struct {
	ImageDir  string
	OutputDir string
	Margin    int
}

type FigureExtraction struct {
	PageNumber   int      `json:"page_number"`
	FigureIndex  int      `json:"figure_index"`
	Description  string   `json:"description"`
	ImagePath    string   `json:"image_path"`
	MarkdownRef  string   `json:"markdown_ref"`
	MarkerSource string   `json:"marker_source,omitempty"`
	CropRect     CropRect `json:"crop_rect"`
	Method       string   `json:"method,omitempty"`
	SidecarPath  string   `json:"sidecar_path,omitempty"`
	DebugPath    string   `json:"debug_path,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
}

type CropRect struct {
	X      int `json:"x" yaml:"x"`
	Y      int `json:"y" yaml:"y"`
	Width  int `json:"width" yaml:"width"`
	Height int `json:"height" yaml:"height"`
}

func EmbedExtractedFigures(markdown string, opts FigureExtractionOptions) (string, []FigureExtraction, error) {
	if strings.TrimSpace(opts.ImageDir) == "" {
		return "", nil, fmt.Errorf("image_dir is required")
	}
	if strings.TrimSpace(opts.OutputDir) == "" {
		return "", nil, fmt.Errorf("output_dir is required")
	}
	if opts.Margin == 0 {
		opts.Margin = 24
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return "", nil, err
	}

	markdown = synthesizeMissingFigureMarkers(markdown)

	var out strings.Builder
	var figures []FigureExtraction
	currentPage := 0
	figureIndex := 0
	for _, line := range strings.Split(markdown, "\n") {
		if match := pageMarkerPattern.FindStringSubmatch(line); len(match) == 2 {
			_, _ = fmt.Sscanf(match[1], "%d", &currentPage)
			figureIndex = 0
			out.WriteString(line)
			out.WriteByte('\n')
			continue
		}
		if match := figureMarkerLinePattern.FindStringSubmatch(strings.TrimSpace(line)); len(match) == 2 && currentPage > 0 {
			figureIndex++
			desc := strings.TrimSpace(match[1])
			figure, err := extractPageFigure(currentPage, figureIndex, desc, opts)
			if err != nil {
				return "", nil, err
			}
			rel := filepath.ToSlash(filepath.Base(filepath.Dir(figure.ImagePath)) + "/" + filepath.Base(figure.ImagePath))
			alt := strings.NewReplacer("[", "(", "]", ")", "\n", " ").Replace(desc)
			fmt.Fprintf(&out, "![%s](%s)\n", alt, rel)
			markerSource := "explicit"
			if strings.HasPrefix(desc, "Full-page diagram showing ") {
				markerSource = "synthesized-or-explicit"
			}
			figure.MarkdownRef = rel
			figure.MarkerSource = markerSource
			figures = append(figures, figure)
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return strings.TrimRight(out.String(), "\n") + "\n", figures, nil
}

func synthesizeMissingFigureMarkers(markdown string) string {
	matches := pageMarkerPattern.FindAllStringSubmatchIndex(markdown, -1)
	if len(matches) == 0 {
		return markdown
	}
	var out strings.Builder
	out.WriteString(markdown[:matches[0][0]])
	for i, match := range matches {
		marker := markdown[match[0]:match[1]]
		start := match[1]
		end := len(markdown)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		pageText := markdown[start:end]
		out.WriteString(marker)
		out.WriteString(addMissingFigureMarkerToPage(pageText))
	}
	return out.String()
}

func addMissingFigureMarkerToPage(pageText string) string {
	if strings.Contains(pageText, "[FIGURE:") || strings.Contains(pageText, "![") {
		return pageText
	}
	lines := strings.Split(pageText, "\n")
	captionIndex := -1
	caption := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		match := figureCaptionLinePattern.FindStringSubmatch(trimmed)
		if len(match) != 2 || strings.Contains(trimmed, "...") {
			continue
		}
		captionIndex = i
		caption = strings.TrimSpace(match[1])
		break
	}
	if captionIndex < 0 || !looksLikeDiagramPage(lines, captionIndex) {
		return pageText
	}
	description := "Full-page diagram showing " + strings.TrimSuffix(caption, ".")
	insert := "[FIGURE: " + description + "]"
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:captionIndex+1]...)
	if captionIndex+1 < len(lines) && strings.TrimSpace(lines[captionIndex+1]) != "" {
		out = append(out, "")
	}
	out = append(out, insert)
	out = append(out, lines[captionIndex+1:]...)
	return strings.Join(out, "\n")
}

func looksLikeDiagramPage(lines []string, captionIndex int) bool {
	nonEmpty := 0
	shortLines := 0
	diagramCueLines := 0
	proseLike := 0
	for i, line := range lines {
		if i == captionIndex {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		nonEmpty++
		words := strings.Fields(trimmed)
		if len(words) <= 5 {
			shortLines++
		}
		if strings.Contains(trimmed, "->") || strings.Contains(trimmed, "<-") || strings.Contains(trimmed, "---") || strings.Contains(trimmed, "- -") || isMostlyUpperOrLabel(trimmed) {
			diagramCueLines++
		}
		if len(words) >= 9 && strings.ContainsAny(trimmed, ".,;:") {
			proseLike++
		}
	}
	if nonEmpty == 0 {
		return false
	}
	if proseLike >= 3 {
		return false
	}
	return diagramCueLines >= 2 || shortLines*2 >= nonEmpty*3
}

func isMostlyUpperOrLabel(s string) bool {
	letters := 0
	upper := 0
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			letters++
			upper++
		} else if r >= 'a' && r <= 'z' {
			letters++
		}
	}
	return letters >= 4 && upper*2 >= letters
}

func extractPageFigure(pageNumber, figureIndex int, desc string, opts FigureExtractionOptions) (FigureExtraction, error) {
	pagePath := filepath.Join(opts.ImageDir, fmt.Sprintf("page_%03d.png", pageNumber))
	file, err := os.Open(pagePath)
	if err != nil {
		return FigureExtraction{}, err
	}
	defer func() { _ = file.Close() }()
	img, _, err := image.Decode(file)
	if err != nil {
		return FigureExtraction{}, err
	}
	crop, rect := cropNonWhite(img, opts.Margin)
	outPath := filepath.Join(opts.OutputDir, fmt.Sprintf("page_%03d_figure_%02d.png", pageNumber, figureIndex))
	outFile, err := os.Create(outPath)
	if err != nil {
		return FigureExtraction{}, err
	}
	defer func() { _ = outFile.Close() }()
	if err := png.Encode(outFile, crop); err != nil {
		return FigureExtraction{}, err
	}
	figure := FigureExtraction{PageNumber: pageNumber, FigureIndex: figureIndex, Description: desc, ImagePath: outPath, CropRect: cropRectFromImageRect(rect), Method: "ink-band-v1", Warnings: figureCropWarnings(img.Bounds(), rect)}
	if err := writeFigureSidecars(img, &figure, opts.OutputDir); err != nil {
		return FigureExtraction{}, err
	}
	return figure, nil
}

func cropNonWhite(img image.Image, margin int) (image.Image, image.Rectangle) {
	bounds := img.Bounds()
	verticalBand, ok := meaningfulInkUnion(img)
	if !ok {
		return img, bounds
	}
	minX, maxX, ok := horizontalInkBounds(img, verticalBand)
	if !ok {
		return img, bounds
	}
	minY := clamp(verticalBand.Min-margin, bounds.Min.Y, bounds.Max.Y)
	maxY := clamp(verticalBand.Max+margin+1, bounds.Min.Y, bounds.Max.Y)
	minX = clamp(minX-margin, bounds.Min.X, bounds.Max.X)
	maxX = clamp(maxX+margin+1, bounds.Min.X, bounds.Max.X)
	rect := image.Rect(minX, minY, maxX, maxY)
	return cropImage(img, rect), rect
}

type intBand struct {
	Min int
	Max int
	Ink int
}

func meaningfulInkUnion(img image.Image) (intBand, bool) {
	bounds := img.Bounds()
	bands := inkBands(img)
	if len(bands) == 0 {
		return intBand{}, false
	}
	topLimit := bounds.Min.Y + bounds.Dy()/25
	bottomLimit := bounds.Max.Y - bounds.Dy()/6
	minHeight := maxInt(12, bounds.Dy()/180)
	var union intBand
	found := false
	for _, band := range bands {
		height := band.Max - band.Min + 1
		if band.Max < topLimit || band.Min > bottomLimit {
			continue
		}
		if height < minHeight && band.Ink < 200 {
			continue
		}
		if !found {
			union = band
			found = true
			continue
		}
		if band.Min < union.Min {
			union.Min = band.Min
		}
		if band.Max > union.Max {
			union.Max = band.Max
		}
		union.Ink += band.Ink
	}
	if found {
		return union, true
	}
	return dominantInkBand(img)
}

func inkBands(img image.Image) []intBand {
	bounds := img.Bounds()
	left := bounds.Min.X + bounds.Dx()/10
	right := bounds.Max.X - bounds.Dx()/20
	rowThreshold := maxInt(8, bounds.Dx()/250)
	bands := []intBand{}
	inBand := false
	current := intBand{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		ink := 0
		for x := left; x < right; x++ {
			if isNonWhite(img.At(x, y)) {
				ink++
			}
		}
		if ink >= rowThreshold {
			if !inBand {
				current = intBand{Min: y, Max: y, Ink: ink}
				inBand = true
			} else {
				current.Max = y
				current.Ink += ink
			}
			continue
		}
		if inBand {
			bands = append(bands, current)
			inBand = false
		}
	}
	if inBand {
		bands = append(bands, current)
	}
	return bands
}

func dominantInkBand(img image.Image) (intBand, bool) {
	bounds := img.Bounds()
	rowThreshold := maxInt(8, bounds.Dx()/250)
	bands := inkBands(img)
	if len(bands) == 0 {
		return intBand{}, false
	}
	best := bands[0]
	for _, band := range bands[1:] {
		bestScore := best.Ink + (best.Max-best.Min)*rowThreshold
		score := band.Ink + (band.Max-band.Min)*rowThreshold
		if score > bestScore {
			best = band
		}
	}
	return best, true
}

func horizontalInkBounds(img image.Image, band intBand) (int, int, bool) {
	bounds := img.Bounds()
	minX, maxX := bounds.Max.X, bounds.Min.X
	found := false
	for y := band.Min; y <= band.Max; y++ {
		for x := bounds.Min.X + bounds.Dx()/12; x < bounds.Max.X-bounds.Dx()/20; x++ {
			if isNonWhite(img.At(x, y)) {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				found = true
			}
		}
	}
	return minX, maxX, found
}

func cropRectFromImageRect(rect image.Rectangle) CropRect {
	return CropRect{X: rect.Min.X, Y: rect.Min.Y, Width: rect.Dx(), Height: rect.Dy()}
}

func figureCropWarnings(page image.Rectangle, crop image.Rectangle) []string {
	warnings := []string{}
	if crop.Dx() <= 0 || crop.Dy() <= 0 {
		return []string{"empty crop"}
	}
	pageArea := page.Dx() * page.Dy()
	cropArea := crop.Dx() * crop.Dy()
	if pageArea > 0 {
		ratio := float64(cropArea) / float64(pageArea)
		if ratio > 0.85 {
			warnings = append(warnings, "crop covers more than 85 percent of page")
		}
		if ratio < 0.02 {
			warnings = append(warnings, "crop covers less than 2 percent of page")
		}
	}
	if crop.Max.Y > page.Max.Y-page.Dy()/10 {
		warnings = append(warnings, "crop extends into bottom page-furniture zone")
	}
	return warnings
}

func writeFigureSidecars(page image.Image, figure *FigureExtraction, outputDir string) error {
	base := strings.TrimSuffix(filepath.Base(figure.ImagePath), filepath.Ext(figure.ImagePath))
	sidecarPath := filepath.Join(outputDir, base+".json")
	debugPath := filepath.Join(outputDir, base+".debug.png")
	figure.SidecarPath = sidecarPath
	figure.DebugPath = debugPath
	body, err := json.MarshalIndent(figure, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(sidecarPath, append(body, '\n'), 0o644); err != nil {
		return err
	}
	return writeDebugOverlay(page, image.Rect(figure.CropRect.X, figure.CropRect.Y, figure.CropRect.X+figure.CropRect.Width, figure.CropRect.Y+figure.CropRect.Height), debugPath)
}

func writeDebugOverlay(page image.Image, rect image.Rectangle, path string) error {
	bounds := page.Bounds()
	out := image.NewRGBA(bounds)
	draw.Draw(out, bounds, page, bounds.Min, draw.Src)
	red := color.RGBA{R: 255, A: 255}
	for x := rect.Min.X; x < rect.Max.X; x++ {
		for dy := 0; dy < 4; dy++ {
			if rect.Min.Y+dy < bounds.Max.Y {
				out.Set(x, rect.Min.Y+dy, red)
			}
			if rect.Max.Y-1-dy >= bounds.Min.Y {
				out.Set(x, rect.Max.Y-1-dy, red)
			}
		}
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for dx := 0; dx < 4; dx++ {
			if rect.Min.X+dx < bounds.Max.X {
				out.Set(rect.Min.X+dx, y, red)
			}
			if rect.Max.X-1-dx >= bounds.Min.X {
				out.Set(rect.Max.X-1-dx, y, red)
			}
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	return png.Encode(file, out)
}

func isNonWhite(c color.Color) bool {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return false
	}
	return r>>8 < 245 || g>>8 < 245 || b>>8 < 245
}

func cropImage(img image.Image, rect image.Rectangle) image.Image {
	if sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	}); ok {
		return sub.SubImage(rect)
	}
	out := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			out.Set(x-rect.Min.X, y-rect.Min.Y, img.At(x, y))
		}
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func init() {
	image.RegisterFormat("png", "\x89PNG\r\n\x1a\n", png.Decode, png.DecodeConfig)
	image.RegisterFormat("jpeg", "\xff\xd8", jpeg.Decode, jpeg.DecodeConfig)
}

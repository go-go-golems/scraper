package ocrquality

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var figureMarkerLinePattern = regexp.MustCompile(`^\[FIGURE:\s*(.*?)\]\s*$`)

type FigureExtractionOptions struct {
	ImageDir  string
	OutputDir string
	Margin    int
}

type FigureExtraction struct {
	PageNumber  int    `json:"page_number"`
	FigureIndex int    `json:"figure_index"`
	Description string `json:"description"`
	ImagePath   string `json:"image_path"`
	MarkdownRef string `json:"markdown_ref"`
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
			imagePath, err := extractPageFigure(currentPage, figureIndex, desc, opts)
			if err != nil {
				return "", nil, err
			}
			rel := filepath.ToSlash(filepath.Base(filepath.Dir(imagePath)) + "/" + filepath.Base(imagePath))
			alt := strings.NewReplacer("[", "(", "]", ")", "\n", " ").Replace(desc)
			fmt.Fprintf(&out, "![%s](%s)\n", alt, rel)
			figures = append(figures, FigureExtraction{PageNumber: currentPage, FigureIndex: figureIndex, Description: desc, ImagePath: imagePath, MarkdownRef: rel})
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return strings.TrimRight(out.String(), "\n") + "\n", figures, nil
}

func extractPageFigure(pageNumber, figureIndex int, desc string, opts FigureExtractionOptions) (string, error) {
	pagePath := filepath.Join(opts.ImageDir, fmt.Sprintf("page_%03d.png", pageNumber))
	file, err := os.Open(pagePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	img, _, err := image.Decode(file)
	if err != nil {
		return "", err
	}
	crop := cropNonWhite(img, opts.Margin)
	outPath := filepath.Join(opts.OutputDir, fmt.Sprintf("page_%03d_figure_%02d.png", pageNumber, figureIndex))
	outFile, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = outFile.Close() }()
	if err := png.Encode(outFile, crop); err != nil {
		return "", err
	}
	return outPath, nil
}

func cropNonWhite(img image.Image, margin int) image.Image {
	bounds := img.Bounds()
	verticalBand, ok := meaningfulInkUnion(img)
	if !ok {
		return img
	}
	minX, maxX, ok := horizontalInkBounds(img, verticalBand)
	if !ok {
		return img
	}
	minY := clamp(verticalBand.Min-margin, bounds.Min.Y, bounds.Max.Y)
	maxY := clamp(verticalBand.Max+margin+1, bounds.Min.Y, bounds.Max.Y)
	minX = clamp(minX-margin, bounds.Min.X, bounds.Max.X)
	maxX = clamp(maxX+margin+1, bounds.Min.X, bounds.Max.X)
	return cropImage(img, image.Rect(minX, minY, maxX, maxY))
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

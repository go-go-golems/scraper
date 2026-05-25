package bookprofile

import (
	"fmt"
	"os"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

type BookFamily string

const (
	FamilyTechnicalReport BookFamily = "technical-report"
	FamilyTextbook        BookFamily = "textbook"
	FamilyNovel           BookFamily = "novel"
	FamilyManual          BookFamily = "manual"
	FamilyMath            BookFamily = "math"
	FamilyMagazine        BookFamily = "magazine"
	FamilyHistoricalScan  BookFamily = "historical-scan"
)

type PageType string

const (
	PageBlank         PageType = "blank"
	PageTitle         PageType = "title"
	PageCopyright     PageType = "copyright"
	PageTOC           PageType = "table-of-contents"
	PageFigureList    PageType = "table-of-figures"
	PageBody          PageType = "body"
	PageDiagram       PageType = "diagram"
	PageTable         PageType = "table"
	PageEquationDense PageType = "equation-dense"
	PageIndex         PageType = "index"
	PageBibliography  PageType = "bibliography"
)

type Profile struct {
	ID          string           `yaml:"id" json:"id"`
	DisplayName string           `yaml:"display_name,omitempty" json:"display_name,omitempty"`
	Family      BookFamily       `yaml:"family,omitempty" json:"family,omitempty"`
	PageImages  PageImagePolicy  `yaml:"page_images,omitempty" json:"page_images,omitempty"`
	Prompt      PromptPolicy     `yaml:"prompt,omitempty" json:"prompt,omitempty"`
	Vocabulary  VocabularyPolicy `yaml:"vocabulary,omitempty" json:"vocabulary,omitempty"`
	PageTypes   PageTypePolicy   `yaml:"page_types,omitempty" json:"page_types,omitempty"`
	QA          QAPolicy         `yaml:"qa,omitempty" json:"qa,omitempty"`
	Normalize   NormalizePolicy  `yaml:"normalize,omitempty" json:"normalize,omitempty"`
	Figures     FigurePolicy     `yaml:"figures,omitempty" json:"figures,omitempty"`
	Context     ContextPolicy    `yaml:"context,omitempty" json:"context,omitempty"`
}

type PageImagePolicy struct {
	Glob            string `yaml:"glob,omitempty" json:"glob,omitempty"`
	PageNumberRegex string `yaml:"page_number_regex,omitempty" json:"page_number_regex,omitempty"`
}

type PromptPolicy struct {
	BaseTemplate          string              `yaml:"base_template,omitempty" json:"base_template,omitempty"`
	FigureMarkerContract  bool                `yaml:"figure_marker_contract,omitempty" json:"figure_marker_contract,omitempty"`
	PageTypeInstructions  map[PageType]string `yaml:"page_type_instructions,omitempty" json:"page_type_instructions,omitempty"`
	PreserveLineBreaksFor []PageType          `yaml:"preserve_line_breaks_for,omitempty" json:"preserve_line_breaks_for,omitempty"`
}

type VocabularyPolicy struct {
	PreferredTerms      map[string]string `yaml:"preferred_terms,omitempty" json:"preferred_terms,omitempty"`
	ProtectedTerms      []string          `yaml:"protected_terms,omitempty" json:"protected_terms,omitempty"`
	HistoricalSpellings []string          `yaml:"historical_spellings,omitempty" json:"historical_spellings,omitempty"`
	KnownBadTerms       []string          `yaml:"known_bad_terms,omitempty" json:"known_bad_terms,omitempty"`
	Names               []string          `yaml:"names,omitempty" json:"names,omitempty"`
}

type PageTypePolicy struct {
	KnownPages map[int]PageType `yaml:"known_pages,omitempty" json:"known_pages,omitempty"`
}

type QAPolicy struct {
	ExpectedPages          int      `yaml:"expected_pages,omitempty" json:"expected_pages,omitempty"`
	KnownBadTerms          []string `yaml:"known_bad_terms,omitempty" json:"known_bad_terms,omitempty"`
	ExpectedStrings        []string `yaml:"expected_strings,omitempty" json:"expected_strings,omitempty"`
	ListPages              []int    `yaml:"list_pages,omitempty" json:"list_pages,omitempty"`
	RequiredFigureCaptions []string `yaml:"required_figure_captions,omitempty" json:"required_figure_captions,omitempty"`
	ExpectedFigureCount    int      `yaml:"expected_figure_count,omitempty" json:"expected_figure_count,omitempty"`
}

type NormalizePolicy struct {
	ListPages  []int `yaml:"list_pages,omitempty" json:"list_pages,omitempty"`
	DotLeaders bool  `yaml:"dot_leaders,omitempty" json:"dot_leaders,omitempty"`
}

type FigurePolicy struct {
	Enabled                  bool     `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	MarkerSyntax             string   `yaml:"marker_syntax,omitempty" json:"marker_syntax,omitempty"`
	CaptionPatterns          []string `yaml:"caption_patterns,omitempty" json:"caption_patterns,omitempty"`
	SynthesizeMissingMarkers bool     `yaml:"synthesize_missing_markers,omitempty" json:"synthesize_missing_markers,omitempty"`
	SegmentationStrategy     string   `yaml:"segmentation_strategy,omitempty" json:"segmentation_strategy,omitempty"`
	OutputRawCrop            bool     `yaml:"output_raw_crop,omitempty" json:"output_raw_crop,omitempty"`
	OutputEnhancedCrop       bool     `yaml:"output_enhanced_crop,omitempty" json:"output_enhanced_crop,omitempty"`
	OutputDebugOverlay       bool     `yaml:"output_debug_overlay,omitempty" json:"output_debug_overlay,omitempty"`
}

type ContextPolicy struct {
	DefaultWindow int        `yaml:"default_window,omitempty" json:"default_window,omitempty"`
	MaxWindow     int        `yaml:"max_window,omitempty" json:"max_window,omitempty"`
	EnableFor     []PageType `yaml:"enable_for,omitempty" json:"enable_for,omitempty"`
}

func Load(path string) (Profile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, err
	}
	var profile Profile
	if err := yaml.Unmarshal(body, &profile); err != nil {
		return Profile{}, err
	}
	if profile.ID == "" {
		return Profile{}, fmt.Errorf("profile id is required")
	}
	return profile, nil
}

func Save(path string, profile Profile) error {
	body, err := yaml.Marshal(profile)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func Resolve(bookID, profilePath string) (Profile, bool, error) {
	if profilePath != "" {
		profile, err := Load(profilePath)
		return profile, true, err
	}
	switch bookID {
	case "report-794", "presentation-based-uis-hq-007-v4-mini-30", "presentation-based-uis":
		return Report794(), true, nil
	default:
		return Profile{}, false, nil
	}
}

func Report794() Profile {
	return Profile{
		ID:          "report-794",
		DisplayName: "MIT Technical Report 794 - Presentation Based User Interfaces",
		Family:      FamilyTechnicalReport,
		PageImages: PageImagePolicy{
			Glob:            "page_*.png",
			PageNumberRegex: `page_(\d+)\.png`,
		},
		Prompt: PromptPolicy{BaseTemplate: "technical-report-v1", FigureMarkerContract: true},
		Vocabulary: VocabularyPolicy{
			ProtectedTerms:      []string{"Presentation Based User Interfaces", "Eugene C. Ciccarelli IV", "Eugene Charles Ciccarelli IV", "PSBase", "PPS", "PPSCalc", "Dired", "Steamer", "Zmacs", "Xerox Star"},
			HistoricalSpellings: []string{"data base"},
			KnownBadTerms:       []string{"DiRed", "Streamer", "PPSBase", "Ciccarrelli", "[IMAGE:"},
			PreferredTerms:      map[string]string{"DiRed": "Dired", "Streamer": "Steamer", "PPSBase": "PSBase", "Ciccarrelli": "Ciccarelli"},
		},
		PageTypes: PageTypePolicy{KnownPages: map[int]PageType{1: PageTitle, 6: PageTOC, 7: PageTOC, 8: PageFigureList, 9: PageFigureList, 13: PageDiagram, 15: PageDiagram, 17: PageDiagram, 21: PageDiagram}},
		QA: QAPolicy{
			ExpectedPages:          30,
			KnownBadTerms:          []string{"DiRed", "Streamer", "PPSBase", "Ciccarrelli", "[IMAGE:"},
			ExpectedStrings:        []string{"Presentation Based User Interfaces", "This blank page was inserted to preserve pagination.", "Figure 4-1: Dired Model", "Figure 4-9: Sample Steamer Schematic", "Figure 5-1: PSBase Support of PPS Components", "Chapter Two", "The Primitive Presentation System (PPS) Model", "2.1 PPSCalc"},
			ListPages:              []int{6, 7, 8, 9},
			RequiredFigureCaptions: []string{"Figure 1-1: A Rudimentary User Interface", "Figure 1-2: The Representation Shift Model", "Figure 1-3: The Primitive Presentation System (PPS) Model", "Figure 1-4: Structure of PSBase"},
			ExpectedFigureCount:    4,
		},
		Normalize: NormalizePolicy{ListPages: []int{6, 7, 8, 9}, DotLeaders: true},
		Figures:   FigurePolicy{Enabled: true, MarkerSyntax: "[FIGURE: ...]", SynthesizeMissingMarkers: true, SegmentationStrategy: "ink-band-v1"},
		Context:   ContextPolicy{DefaultWindow: 0, MaxWindow: 2, EnableFor: []PageType{PageTOC, PageFigureList, PageBody, PageDiagram}},
	}
}

func ListPages(profile Profile) []int {
	if len(profile.QA.ListPages) > 0 {
		return append([]int(nil), profile.QA.ListPages...)
	}
	return append([]int(nil), profile.Normalize.ListPages...)
}

func KnownPageTypes(profile Profile) []int {
	pages := make([]int, 0, len(profile.PageTypes.KnownPages))
	for page := range profile.PageTypes.KnownPages {
		pages = append(pages, page)
	}
	sort.Ints(pages)
	return pages
}

func NowTimestamp() string { return time.Now().Format(time.RFC3339) }

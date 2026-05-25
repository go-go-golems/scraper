package ocrquality

const (
	PackageName = "ocr-quality"

	KindQABefore       = "ocr-quality/qa-before"
	KindNormalize      = "ocr-quality/normalize-markdown"
	KindQAAfter        = "ocr-quality/qa-after"
	KindImportLog      = "ocr-quality/import-log"
	KindEmbedFigures   = "ocr-quality/embed-figures"
	KindAssembleReport = "ocr-quality/assemble-report"

	QueueQuality = "ocr-quality"
)

type RunInput struct {
	BookID          string   `json:"book_id,omitempty"`
	MarkdownPath    string   `json:"markdown_path"`
	OutputDir       string   `json:"output_dir,omitempty"`
	ExpectedPages   int      `json:"expected_pages,omitempty"`
	KnownBadTerms   []string `json:"known_bad_terms,omitempty"`
	ExpectedStrings []string `json:"expected_strings,omitempty"`
	ListPages       []int    `json:"list_pages,omitempty"`
	LogPath         string   `json:"log_path,omitempty"`
	ImageDir        string   `json:"image_dir,omitempty"`
	EmbedFigures    bool     `json:"embed_figures,omitempty"`
}

type QAInput struct {
	MarkdownPath    string   `json:"markdown_path"`
	ExpectedPages   int      `json:"expected_pages,omitempty"`
	KnownBadTerms   []string `json:"known_bad_terms,omitempty"`
	ExpectedStrings []string `json:"expected_strings,omitempty"`
	ListPages       []int    `json:"list_pages,omitempty"`
	ReportName      string   `json:"report_name,omitempty"`
}

type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type QAFinding struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Page     int      `json:"page,omitempty"`
	Line     int      `json:"line,omitempty"`
}

type PageStyleCheck struct {
	Page                 int      `json:"page"`
	Label                string   `json:"label"`
	MarkdownBulletLines  int      `json:"markdown_bullet_lines"`
	MarkdownHeadingLines int      `json:"markdown_heading_lines"`
	BulletSamples        []string `json:"bullet_samples,omitempty"`
	HeadingSamples       []string `json:"heading_samples,omitempty"`
}

type QAResult struct {
	MarkdownPath       string           `json:"markdown_path"`
	ExpectedPages      int              `json:"expected_pages"`
	PageMarkersFound   int              `json:"page_markers_found"`
	ObservedPages      []int            `json:"observed_pages"`
	FigureMarkers      int              `json:"figure_markers"`
	KnownBadTermHits   map[string]int   `json:"known_bad_term_hits,omitempty"`
	ExpectedStringHits map[string]bool  `json:"expected_string_hits,omitempty"`
	DuplicateLines     []QAFinding      `json:"duplicate_lines,omitempty"`
	ListPageChecks     []PageStyleCheck `json:"list_page_checks,omitempty"`
	Findings           []QAFinding      `json:"findings,omitempty"`
	Passed             bool             `json:"passed"`
	ReportMarkdown     string           `json:"report_markdown,omitempty"`
	ReportRefID        string           `json:"report_ref_id,omitempty"`
	ReportRefURI       string           `json:"report_ref_uri,omitempty"`
}

type NormalizeInput struct {
	MarkdownPath string `json:"markdown_path"`
	OutputPath   string `json:"output_path,omitempty"`
	DiffPath     string `json:"diff_path,omitempty"`
	ListPages    []int  `json:"list_pages,omitempty"`
}

type NormalizeResult struct {
	InputPath       string `json:"input_path"`
	OutputPath      string `json:"output_path"`
	DiffPath        string `json:"diff_path,omitempty"`
	Changed         bool   `json:"changed"`
	ChangedLines    int    `json:"changed_lines"`
	ChangedPages    []int  `json:"changed_pages,omitempty"`
	OutputRefID     string `json:"output_ref_id,omitempty"`
	OutputRefURI    string `json:"output_ref_uri,omitempty"`
	DiffRefID       string `json:"diff_ref_id,omitempty"`
	DiffRefURI      string `json:"diff_ref_uri,omitempty"`
	NormalizedBytes int    `json:"normalized_bytes"`
}

type EmbedFiguresInput struct {
	MarkdownPath string `json:"markdown_path"`
	ImageDir     string `json:"image_dir"`
	OutputPath   string `json:"output_path,omitempty"`
	FiguresDir   string `json:"figures_dir,omitempty"`
}

type EmbedFiguresResult struct {
	InputPath      string             `json:"input_path"`
	OutputPath     string             `json:"output_path"`
	FiguresDir     string             `json:"figures_dir"`
	FigureCount    int                `json:"figure_count"`
	Figures        []FigureExtraction `json:"figures,omitempty"`
	OutputRefID    string             `json:"output_ref_id,omitempty"`
	OutputRefURI   string             `json:"output_ref_uri,omitempty"`
	FigureImageIDs []string           `json:"figure_image_ids,omitempty"`
}

type LogImportInput struct {
	LogPath    string `json:"log_path"`
	SQLitePath string `json:"sqlite_path,omitempty"`
	ReportName string `json:"report_name,omitempty"`
}

type LogImportResult struct {
	LogPath            string         `json:"log_path"`
	SQLitePath         string         `json:"sqlite_path,omitempty"`
	TotalLines         int            `json:"total_lines"`
	ParsedLines        int            `json:"parsed_lines"`
	LevelCounts        map[string]int `json:"level_counts,omitempty"`
	TraceLines         int            `json:"trace_lines"`
	WarningErrorLines  int            `json:"warning_error_lines"`
	NonTraceEventLines int            `json:"non_trace_event_lines"`
	ReportMarkdown     string         `json:"report_markdown,omitempty"`
	ReportRefID        string         `json:"report_ref_id,omitempty"`
	ReportRefURI       string         `json:"report_ref_uri,omitempty"`
}

type ReportInput struct {
	BookID              string `json:"book_id,omitempty"`
	RawMarkdownPath     string `json:"raw_markdown_path"`
	NormalizedPath      string `json:"normalized_path,omitempty"`
	EmbeddedPath        string `json:"embedded_path,omitempty"`
	BeforeQARefURI      string `json:"before_qa_ref_uri,omitempty"`
	AfterQARefURI       string `json:"after_qa_ref_uri,omitempty"`
	NormalizeDiffRefURI string `json:"normalize_diff_ref_uri,omitempty"`
}

type ReportResult struct {
	ReportRefID  string `json:"report_ref_id"`
	ReportRefURI string `json:"report_ref_uri"`
}

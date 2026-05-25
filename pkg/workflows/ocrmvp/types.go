package ocrmvp

import "context"

const (
	PackageName          = "ocr-mvp"
	ProjectionName       = "ocr-mvp"
	KindDiscoverPages    = "ocr-mvp/discover-pages"
	KindOCRPage          = "ocr-mvp/ocr-page"
	KindAssembleMarkdown = "ocr-mvp/assemble-markdown"

	QueueControl  = "ocr-control"
	QueueOCR      = "ocr"
	QueueAssemble = "ocr-assemble"
)

type RunInput struct {
	BookID            string   `json:"book_id"`
	ImageDir          string   `json:"image_dir"`
	PageGlob          string   `json:"page_glob,omitempty"`
	StartPage         int      `json:"start_page,omitempty"`
	EndPage           int      `json:"end_page,omitempty"`
	Profile           string   `json:"profile,omitempty"`
	ProfileRegistries []string `json:"profile_registries,omitempty"`
	PromptVersion     string   `json:"prompt_version,omitempty"`
	ContextWindow     int      `json:"context_window,omitempty"`
	DryRun            bool     `json:"dry_run,omitempty"`
}

type PageSpec struct {
	BookID     string `json:"book_id"`
	PageNumber int    `json:"page_number"`
	ImagePath  string `json:"image_path"`
}

type DiscoverResult struct {
	BookID     string     `json:"book_id"`
	PageCount  int        `json:"page_count"`
	OCRStepIDs []string   `json:"ocr_step_ids"`
	Pages      []PageSpec `json:"pages"`
}

type PageContextImage struct {
	PageNumber int    `json:"page_number"`
	ImagePath  string `json:"image_path"`
	Relation   string `json:"relation"`
}

type PageOCRInput struct {
	BookID            string             `json:"book_id"`
	PageNumber        int                `json:"page_number"`
	ImagePath         string             `json:"image_path"`
	Profile           string             `json:"profile,omitempty"`
	ProfileRegistries []string           `json:"profile_registries,omitempty"`
	PromptVersion     string             `json:"prompt_version"`
	ContextBefore     []PageContextImage `json:"context_before,omitempty"`
	ContextAfter      []PageContextImage `json:"context_after,omitempty"`
	DryRun            bool               `json:"dry_run,omitempty"`
}

type PageOCRResult struct {
	BookID         string `json:"book_id"`
	PageNumber     int    `json:"page_number"`
	Markdown       string `json:"markdown,omitempty"`
	MarkdownRefID  string `json:"markdown_ref_id"`
	MarkdownRefURI string `json:"markdown_ref_uri"`
	CharCount      int    `json:"char_count"`
	PromptVersion  string `json:"prompt_version"`
	Profile        string `json:"profile,omitempty"`
	Registry       string `json:"registry,omitempty"`
}

type AssembleInput struct {
	BookID string `json:"book_id"`
}

type AssembleResult struct {
	BookID        string `json:"book_id"`
	PageCount     int    `json:"page_count"`
	MarkdownRefID string `json:"markdown_ref_id"`
	MarkdownURI   string `json:"markdown_uri"`
	CharCount     int    `json:"char_count"`
}

type OCRTextResult struct {
	Text             string            `json:"text"`
	ProfileSlug      string            `json:"profile_slug,omitempty"`
	RegistrySlug     string            `json:"registry_slug,omitempty"`
	ConfigFiles      []string          `json:"config_files,omitempty"`
	PromptVersion    string            `json:"prompt_version,omitempty"`
	ProviderMetadata map[string]string `json:"provider_metadata,omitempty"`
}

type OCRClient interface {
	OCRPage(ctx context.Context, input PageOCRInput, imageBytes []byte) (OCRTextResult, error)
}

type OCRClientFunc func(ctx context.Context, input PageOCRInput, imageBytes []byte) (OCRTextResult, error)

func (f OCRClientFunc) OCRPage(ctx context.Context, input PageOCRInput, imageBytes []byte) (OCRTextResult, error) {
	return f(ctx, input, imageBytes)
}

package ocrmvp

import (
	"context"
	"fmt"
)

type DryRunOCRClient struct{}

func (DryRunOCRClient) OCRPage(ctx context.Context, input PageOCRInput, imageBytes []byte) (OCRTextResult, error) {
	select {
	case <-ctx.Done():
		return OCRTextResult{}, ctx.Err()
	default:
	}
	text := fmt.Sprintf("## Page %03d\n\nDry-run OCR for book `%s` from `%s` (%d image bytes).", input.PageNumber, input.BookID, input.ImagePath, len(imageBytes))
	return OCRTextResult{
		Text:          text,
		ProfileSlug:   input.Profile,
		PromptVersion: normalizePromptVersion(input.PromptVersion),
		ProviderMetadata: map[string]string{
			"mode": "dry-run",
		},
	}, nil
}

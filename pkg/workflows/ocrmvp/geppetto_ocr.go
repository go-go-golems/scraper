package ocrmvp

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/pinocchio/pkg/cmds/profilebootstrap"
)

type GeppettoOCRClient struct{}

func NewGeppettoOCRClient() *GeppettoOCRClient {
	return &GeppettoOCRClient{}
}

func (c *GeppettoOCRClient) OCRPage(ctx context.Context, input PageOCRInput, imageBytes []byte) (OCRTextResult, error) {
	parsed, err := newPinocchioSelectionValues(input)
	if err != nil {
		return OCRTextResult{}, fmt.Errorf("build pinocchio profile selection: %w", err)
	}
	resolved, err := profilebootstrap.ResolveCLIEngineSettings(ctx, parsed)
	if err != nil {
		return OCRTextResult{}, fmt.Errorf("resolve pinocchio profile-backed engine settings: %w", err)
	}
	if resolved.Close != nil {
		defer resolved.Close()
	}
	eng, err := profilebootstrap.NewEngineFromResolvedCLIEngineSettings(resolved)
	if err != nil {
		return OCRTextResult{}, fmt.Errorf("build geppetto engine from pinocchio profile: %w", err)
	}

	turn := &turns.Turn{}
	turns.AppendBlock(turn, turns.NewSystemTextBlock(OCRSystemPrompt))
	images, err := multimodalImages(input, imageBytes)
	if err != nil {
		return OCRTextResult{}, err
	}
	turns.AppendBlock(turn, turns.NewUserMultimodalBlock(RenderPagePrompt(input), images))
	updated, err := eng.RunInference(ctx, turn)
	if err != nil {
		return OCRTextResult{}, fmt.Errorf("run geppetto OCR inference: %w", err)
	}
	text, err := lastLLMText(updated)
	if err != nil {
		return OCRTextResult{}, err
	}
	result := OCRTextResult{
		Text:          strings.TrimSpace(text),
		PromptVersion: normalizePromptVersion(input.PromptVersion),
		ConfigFiles:   append([]string(nil), resolved.ConfigFiles...),
	}
	if resolved.ResolvedEngineProfile != nil {
		result.ProfileSlug = resolved.ResolvedEngineProfile.EngineProfileSlug.String()
		result.RegistrySlug = resolved.ResolvedEngineProfile.RegistrySlug.String()
	}
	return result, nil
}

func multimodalImages(input PageOCRInput, targetImageBytes []byte) ([]map[string]any, error) {
	images := []map[string]any{{
		"media_type": mediaTypeFromPath(input.ImagePath),
		"content":    append([]byte(nil), targetImageBytes...),
		"detail":     "high",
		"role":       "target",
		"page":       input.PageNumber,
	}}
	for _, ctxImage := range append(append([]PageContextImage(nil), input.ContextBefore...), input.ContextAfter...) {
		body, err := os.ReadFile(ctxImage.ImagePath)
		if err != nil {
			return nil, fmt.Errorf("read %s context image for page %03d: %w", ctxImage.Relation, ctxImage.PageNumber, err)
		}
		images = append(images, map[string]any{
			"media_type": mediaTypeFromPath(ctxImage.ImagePath),
			"content":    body,
			"detail":     "high",
			"role":       ctxImage.Relation,
			"page":       ctxImage.PageNumber,
		})
	}
	return images, nil
}

func newPinocchioSelectionValues(input PageOCRInput) (*values.Values, error) {
	return profilebootstrap.NewCLISelectionValues(profilebootstrap.CLISelectionInput{
		Profile:           input.Profile,
		ProfileRegistries: append([]string(nil), input.ProfileRegistries...),
	})
}

func mediaTypeFromPath(path string) string {
	if mediaType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); mediaType != "" {
		return mediaType
	}
	return "application/octet-stream"
}

func lastLLMText(turn *turns.Turn) (string, error) {
	if turn == nil {
		return "", fmt.Errorf("nil geppetto turn")
	}
	blocks := turns.FindLastBlocksByKind(*turn, turns.BlockKindLLMText)
	if len(blocks) == 0 {
		return "", fmt.Errorf("geppetto OCR response did not include an LLM text block")
	}
	text, _ := blocks[len(blocks)-1].Payload[turns.PayloadKeyText].(string)
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("geppetto OCR response was empty")
	}
	return text, nil
}

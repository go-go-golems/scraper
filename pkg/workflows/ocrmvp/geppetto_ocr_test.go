package ocrmvp

import (
	"context"
	"os"
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
	"github.com/go-go-golems/pinocchio/pkg/cmds/profilebootstrap"
	"github.com/stretchr/testify/require"
)

func TestLastLLMText(t *testing.T) {
	turn := &turns.Turn{}
	turns.AppendBlock(turn, turns.NewUserTextBlock("prompt"))
	turns.AppendBlock(turn, turns.NewAssistantTextBlock(" first "))
	turns.AppendBlock(turn, turns.NewAssistantTextBlock(" final markdown "))

	text, err := lastLLMText(turn)
	require.NoError(t, err)
	require.Equal(t, "final markdown", text)
}

func TestLastLLMTextRejectsMissingOutput(t *testing.T) {
	turn := &turns.Turn{}
	turns.AppendBlock(turn, turns.NewUserTextBlock("prompt"))
	_, err := lastLLMText(turn)
	require.Error(t, err)
	require.Contains(t, err.Error(), "LLM text block")
}

func TestMediaTypeFromPath(t *testing.T) {
	require.Equal(t, "image/png", mediaTypeFromPath("page_001.png"))
	require.Equal(t, "image/jpeg", mediaTypeFromPath("page_001.jpg"))
	require.Equal(t, "application/octet-stream", mediaTypeFromPath("page_001.unknownext"))
}

func TestPinocchioSelectionValuesPreserveProfileAndRegistries(t *testing.T) {
	parsed, err := newPinocchioSelectionValues(PageOCRInput{
		Profile:           "gpt-5-nano-low",
		ProfileRegistries: []string{"profiles-a.yaml", "profiles-b.yaml"},
	})
	require.NoError(t, err)
	settings := profilebootstrap.ProfileSettings{}
	require.NoError(t, parsed.DecodeSectionInto(profilebootstrap.ProfileSettingsSectionSlug, &settings))
	require.Equal(t, "gpt-5-nano-low", settings.Profile)
	require.Equal(t, []string{"profiles-a.yaml", "profiles-b.yaml"}, settings.ProfileRegistries)
}

func TestLiveGeppettoOCRClientGuarded(t *testing.T) {
	if os.Getenv("OCR_MVP_LIVE") != "1" {
		t.Skip("set OCR_MVP_LIVE=1 and OCR_MVP_LIVE_IMAGE=/path/to/page.png to run the live Geppetto OCR smoke test")
	}
	imagePath := os.Getenv("OCR_MVP_LIVE_IMAGE")
	if imagePath == "" {
		t.Fatal("OCR_MVP_LIVE_IMAGE is required when OCR_MVP_LIVE=1")
	}
	imageBytes, err := os.ReadFile(imagePath)
	require.NoError(t, err)
	result, err := NewGeppettoOCRClient().OCRPage(context.Background(), PageOCRInput{
		BookID:        "live-smoke",
		PageNumber:    1,
		ImagePath:     imagePath,
		Profile:       os.Getenv("OCR_MVP_LIVE_PROFILE"),
		PromptVersion: DefaultPromptVersion,
	}, imageBytes)
	require.NoError(t, err)
	require.NotEmpty(t, result.Text)
}

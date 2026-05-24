package ocrmvp

import (
	"testing"

	"github.com/go-go-golems/geppetto/pkg/turns"
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

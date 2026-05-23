package runtimestream

import (
	"testing"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"github.com/stretchr/testify/require"
)

func TestRuntimeEventSessionIDs(t *testing.T) {
	require.Equal(t, []string{"runtime:global"}, stringifySessionIDs(RuntimeEventSessionIDs(nil)))
	require.Equal(t, []string{"runtime:global"}, stringifySessionIDs(RuntimeEventSessionIDs(&runtimev1.RuntimeEventV1{})))
	require.Equal(t, []string{"runtime:global", "workflow:wf-1"}, stringifySessionIDs(RuntimeEventSessionIDs(&runtimev1.RuntimeEventV1{WorkflowId: "wf-1"})))
}

func stringifySessionIDs(ids []sessionstream.SessionId) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, string(id))
	}
	return out
}

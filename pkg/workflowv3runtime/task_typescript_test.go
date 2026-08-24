package workflowv3runtime

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaskTypeScriptMatchesExactGolden(t *testing.T) {
	expected, err := os.ReadFile("testdata/workflow-task.d.ts")
	require.NoError(t, err)
	declaration := TaskTypeScript()
	require.Equal(t, string(expected), declaration)
	for _, fragment := range []string{
		`declare module "workflow/task"`,
		"operationKey: string",
		"putJSON(",
		"Promise<OutputRef>",
	} {
		require.Contains(t, declaration, fragment)
	}
	require.NotContains(t, declaration, ": any")
}

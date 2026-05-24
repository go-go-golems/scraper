package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/stretchr/testify/require"
)

type exampleInput struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

func TestTypedExecutorAdapterBuildsOpResult(t *testing.T) {
	exec := NewTypedExecutor("example/echo", func(ctx context.Context, step *StepContext, input exampleInput) error {
		require.Equal(t, "hello", input.Message)
		require.Equal(t, 3, input.Count)

		require.NoError(t, step.Result(map[string]any{"ok": true, "message": input.Message}))
		require.NoError(t, step.Record("pages", "book:001", map[string]any{"status": "done"}))
		artifactID, err := step.Artifact("page_001.md", "text/markdown", []byte("# hello"), ArtifactKind("ocr-markdown"), ArtifactMetadata(map[string]string{"page": "1"}))
		require.NoError(t, err)
		require.Equal(t, model.ArtifactID("wf-1:step-1:artifact:001"), artifactID)

		childID, err := step.Emit("wf-1:child", map[string]any{"child": true}, StepOpts{
			Kind:      "example/child",
			Queue:     "test:child",
			DedupKey:  "child:1",
			DependsOn: []model.Dependency{{OpID: "wf-1:step-1", Required: true}},
			Metadata:  map[string]string{"script": "child"},
		})
		require.NoError(t, err)
		require.Equal(t, model.OpID("wf-1:child"), childID)
		return nil
	})

	input, err := json.Marshal(exampleInput{Message: "hello", Count: 3})
	require.NoError(t, err)

	r := ToRunner(exec)
	result, err := r.Run(context.Background(), runner.RunContext{
		Workflow: model.WorkflowRun{ID: "wf-1", Site: "book-ocr", Name: "Book OCR"},
		Op: model.OpSpec{
			ID:         "wf-1:step-1",
			WorkflowID: "wf-1",
			Site:       "book-ocr",
			Kind:       "example/echo",
			Queue:      "test",
			Input:      input,
		},
		Lease: model.Lease{WorkerID: "worker-1", Token: "token", AcquiredAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)},
		Now:   time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, model.OpID("wf-1:step-1"), result.OpID)
	require.JSONEq(t, `{"ok":true,"message":"hello"}`, string(result.Data))
	require.Len(t, result.Records, 1)
	require.Equal(t, "pages", result.Records[0].Collection)
	require.Equal(t, "book:001", result.Records[0].Key)
	require.JSONEq(t, `{"status":"done"}`, string(result.Records[0].Data))
	require.Len(t, result.Artifacts, 1)
	require.Equal(t, model.ArtifactID("wf-1:step-1:artifact:001"), result.Artifacts[0].ID)
	require.Equal(t, "ocr-markdown", result.Artifacts[0].Kind)
	require.Equal(t, "1", result.Artifacts[0].Metadata["page"])
	require.Equal(t, []byte("# hello"), result.Artifacts[0].Body)
	require.Len(t, result.Emitted, 1)
	require.Equal(t, model.OpID("wf-1:child"), result.Emitted[0].ID)
	require.Equal(t, model.OpID("wf-1:step-1"), *result.Emitted[0].ParentID)
	require.Equal(t, model.SiteName("book-ocr"), result.Emitted[0].Site)
	require.Equal(t, "example/child", result.Emitted[0].Kind)
	require.Equal(t, model.QueueKey("test:child"), result.Emitted[0].Queue)
	require.Equal(t, []model.OpID{"wf-1:child"}, result.EmittedIDs)
}

func TestRegistryRejectsDuplicateExecutorKind(t *testing.T) {
	registry := NewRegistry()
	require.NoError(t, registry.Register(NewExecutor("example", func(context.Context, *StepContext) error { return nil })))
	err := registry.Register(NewExecutor("example", func(context.Context, *StepContext) error { return nil }))
	require.Error(t, err)
	require.Contains(t, err.Error(), "already registered")
	require.Equal(t, []string{"example"}, registry.Kinds())
}

func TestWorkflowErrorCarriesOpError(t *testing.T) {
	cause := errors.New("provider unavailable")
	err := Retryable("provider_unavailable", cause)
	var wfErr *Error
	require.ErrorAs(t, err, &wfErr)
	opErr := wfErr.OpError()
	require.Equal(t, "provider_unavailable", opErr.Code)
	require.Equal(t, "provider unavailable", opErr.Message)
	require.True(t, opErr.Retryable)
	require.False(t, opErr.OccurredAt.IsZero())
}

func TestStepContextInputReportsDecodeError(t *testing.T) {
	exec := NewTypedExecutor("bad", func(ctx context.Context, step *StepContext, input exampleInput) error { return nil })
	_, err := ToRunner(exec).Run(context.Background(), runner.RunContext{
		Workflow: model.WorkflowRun{ID: "wf-1"},
		Op:       model.OpSpec{ID: "op-1", WorkflowID: "wf-1", Input: json.RawMessage(`{"count":"not-an-int"}`)},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode workflow step input")
}

package engineview

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/stretchr/testify/require"
)

func TestListWorkflowArtifacts(t *testing.T) {
	ctx := context.Background()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	store, err := sqlitestore.Open(ctx, engineDB)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	now := time.Date(2026, 4, 7, 19, 0, 0, 0, time.UTC)
	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:        "wf-artifacts",
			Site:      "js-demo",
			Name:      "artifact workflow",
			Status:    model.WorkflowStatusRunning,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Initial: []model.OpSpec{
			{ID: "wf-artifacts:fetch", WorkflowID: "wf-artifacts", Site: "js-demo", Kind: "http/fetch", Queue: "site:js-demo:http", DedupKey: "fetch"},
			{ID: "wf-artifacts:extract", WorkflowID: "wf-artifacts", Site: "js-demo", Kind: "js", Queue: "site:js-demo:js", DedupKey: "extract"},
		},
	}))
	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:        "wf-other",
			Site:      "js-demo",
			Name:      "other workflow",
			Status:    model.WorkflowStatusRunning,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Initial: []model.OpSpec{
			{ID: "wf-other:fetch", WorkflowID: "wf-other", Site: "js-demo", Kind: "http/fetch", Queue: "site:js-demo:http", DedupKey: "other"},
		},
	}))

	completeLeasedOpWithArtifacts(t, ctx, store, "js-demo", "site:js-demo:http", now, map[string]string{
		"name":        "frontpage.html",
		"kind":        "http-response-body",
		"body":        "<html>frontpage</html>",
		"contentType": "text/html",
	})
	completeLeasedOpWithArtifacts(t, ctx, store, "js-demo", "site:js-demo:js", now.Add(time.Second), map[string]string{
		"name":        "summary.json",
		"kind":        "json-output",
		"body":        `{"stories":30}`,
		"contentType": "application/json",
	})
	completeLeasedOpWithArtifacts(t, ctx, store, "js-demo", "site:js-demo:http", now.Add(2*time.Second), map[string]string{
		"name":        "other.html",
		"kind":        "http-response-body",
		"body":        "<html>other</html>",
		"contentType": "text/html",
	})

	service := NewService(engineDB)
	result, err := service.ListWorkflowArtifacts(ctx, "wf-artifacts", ListWorkflowArtifactsOptions{})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, model.WorkflowID("wf-artifacts"), result.WorkflowID)
	require.Equal(t, 2, result.Total)
	require.Len(t, result.Artifacts, 2)
	require.Equal(t, model.OpID("wf-artifacts:fetch"), result.Artifacts[0].OpID)
	require.Equal(t, "frontpage.html", result.Artifacts[0].Name)
	require.True(t, result.Artifacts[0].Previewable)
	require.Equal(t, "html", result.Artifacts[0].PreviewKind)
	require.Equal(t, model.OpID("wf-artifacts:extract"), result.Artifacts[1].OpID)
	require.Equal(t, "summary.json", result.Artifacts[1].Name)
	require.True(t, result.Artifacts[1].Previewable)
	require.Equal(t, "json", result.Artifacts[1].PreviewKind)

	filtered, err := service.ListWorkflowArtifacts(ctx, "wf-artifacts", ListWorkflowArtifactsOptions{
		OpID:   "wf-artifacts:extract",
		Search: "summary",
	})
	require.NoError(t, err)
	require.NotNil(t, filtered)
	require.Equal(t, 1, filtered.Total)
	require.Len(t, filtered.Artifacts, 1)
	require.Equal(t, "summary.json", filtered.Artifacts[0].Name)

	missing, err := service.ListWorkflowArtifacts(ctx, "wf-missing", ListWorkflowArtifactsOptions{})
	require.NoError(t, err)
	require.Nil(t, missing)
}

func TestGetOpResult(t *testing.T) {
	ctx := context.Background()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	store, err := sqlitestore.Open(ctx, engineDB)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	now := time.Date(2026, 4, 7, 19, 30, 0, 0, time.UTC)
	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:        "wf-results",
			Site:      "js-demo",
			Name:      "result workflow",
			Status:    model.WorkflowStatusRunning,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Initial: []model.OpSpec{
			{ID: "wf-results:done", WorkflowID: "wf-results", Site: "js-demo", Kind: "js", Queue: "site:js-demo:js", DedupKey: "done"},
			{ID: "wf-results:pending", WorkflowID: "wf-results", Site: "js-demo", Kind: "js", Queue: "site:js-demo:js", DedupKey: "pending"},
		},
	}))

	op, lease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         "site:js-demo:js",
		Site:          "js-demo",
		LeaseDuration: 30 * time.Second,
		Now:           now,
	})
	require.NoError(t, err)
	require.NotNil(t, op)
	require.Equal(t, model.OpID("wf-results:done"), op.ID)
	require.NoError(t, store.CompleteOp(ctx, op.ID, storecontract.Completion{
		Lease: *lease,
		Result: model.OpResult{
			OpID:        op.ID,
			Data:        []byte(`{"ok":true}`),
			CompletedAt: now,
		},
	}))

	service := NewService(engineDB)
	result, exists, err := service.GetOpResult(ctx, "wf-results", "wf-results:done")
	require.NoError(t, err)
	require.True(t, exists)
	require.NotNil(t, result)
	require.JSONEq(t, `{"ok":true}`, string(result.Data))

	pendingResult, pendingExists, err := service.GetOpResult(ctx, "wf-results", "wf-results:pending")
	require.NoError(t, err)
	require.True(t, pendingExists)
	require.Nil(t, pendingResult)

	missingResult, missingExists, err := service.GetOpResult(ctx, "wf-results", "wf-results:missing")
	require.NoError(t, err)
	require.False(t, missingExists)
	require.Nil(t, missingResult)
}

func completeLeasedOpWithArtifacts(
	t *testing.T,
	ctx context.Context,
	store *sqlitestore.Store,
	site model.SiteName,
	queue model.QueueKey,
	now time.Time,
	artifact map[string]string,
) {
	t.Helper()

	op, lease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         queue,
		Site:          site,
		LeaseDuration: 30 * time.Second,
		Now:           now,
	})
	require.NoError(t, err)
	require.NotNil(t, op)
	require.NotNil(t, lease)
	require.NoError(t, store.CompleteOp(ctx, op.ID, storecontract.Completion{
		Lease: *lease,
		Result: model.OpResult{
			OpID: op.ID,
			Artifacts: []model.ArtifactWrite{
				{
					ID:          model.ArtifactID(string(op.ID) + ":artifact"),
					Name:        artifact["name"],
					Kind:        artifact["kind"],
					ContentType: artifact["contentType"],
					Body:        []byte(artifact["body"]),
					Metadata: map[string]string{
						"source": "test",
					},
				},
			},
			CompletedAt: now,
		},
	}))
}

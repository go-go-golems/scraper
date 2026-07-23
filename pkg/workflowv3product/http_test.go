package workflowv3product_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/scraper/pkg/taskpackages/cookbooklinear"
	"github.com/go-go-golems/scraper/pkg/workflowv3observations"
	"github.com/go-go-golems/scraper/pkg/workflowv3product"
	"github.com/stretchr/testify/require"
)

func TestProductHTTPReadModelsAndCancellation(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root := t.TempDir()
	app, err := workflowv3product.Open(ctx, productConfig(root))
	require.NoError(t, err)
	defer func() { _ = app.Close() }()
	authored, err := app.Authoring.Author(ctx, cookbooklinear.WorkflowSource())
	require.NoError(t, err)
	inputPath := filepath.Join(root, "customers.jsonl")
	require.NoError(t, os.WriteFile(inputPath, []byte("{\"id\":\"1\",\"email\":\"a@example.com\"}\n"), 0o600))
	_, err = app.Submit(ctx, authored.Plan, map[string]workflowv3product.StagedInput{
		"source": {Path: inputPath, Schema: "customer-jsonl-ref/v1"},
	}, root, "api-run")
	require.NoError(t, err)

	handler, err := workflowv3product.NewHTTPHandler(app, workflowv3product.HTTPOptions{OperatorToken: "test-token"})
	require.NoError(t, err)
	server := httptest.NewServer(handler)
	defer server.Close()

	response, err := http.Get(server.URL + "/api/v3/workflow/runs?limit=10")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var runs []workflowv3product.RunSummary
	require.NoError(t, json.NewDecoder(response.Body).Decode(&runs))
	require.NoError(t, response.Body.Close())
	require.Len(t, runs, 1)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/v3/workflow/runs/api-run/cancel", nil)
	require.NoError(t, err)
	response, err = http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, response.StatusCode)
	require.NoError(t, response.Body.Close())

	request, err = http.NewRequest(http.MethodPost, server.URL+"/api/v3/workflow/runs/api-run/cancel", nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer test-token")
	response, err = http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, response.Body.Close())

	response, err = http.Get(server.URL + "/api/v3/workflow/runs/api-run")
	require.NoError(t, err)
	var view workflowv3product.RunView
	require.NoError(t, json.NewDecoder(response.Body).Decode(&view))
	require.NoError(t, response.Body.Close())
	require.Equal(t, "canceled", view.Snapshot.Status)

	response, err = http.Get(server.URL + "/api/v3/workflow/runs/api-run/observations")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	var observations workflowv3observations.ObservationSet
	require.NoError(t, json.NewDecoder(response.Body).Decode(&observations))
	require.NoError(t, response.Body.Close())
	require.Equal(t, workflowv3observations.SchemaVersion, observations.SchemaVersion)
	require.Equal(t, "canceled", observations.RunStatus)

	response, err = http.Get(server.URL + "/api/v3/workflow/task-packages")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, response.Body.Close())
}

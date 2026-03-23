package jsdemo

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	sitemigrate "github.com/go-go-golems/scraper/pkg/sites/migrate"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestJSDemoWorkflow(t *testing.T) {
	ctx := context.Background()
	registry := siteregistry.New()
	require.NoError(t, Register(registry))

	sitesDir := t.TempDir()
	manager := sitemigrate.NewManager(registry)
	report, err := manager.Migrate(ctx, model.SiteName("js-demo"), sitesDir)
	require.NoError(t, err)

	siteDB, err := sql.Open("sqlite3", report.DatabasePath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, siteDB.Close()) })

	engineStore, err := sqlitestore.Open(ctx, filepath.Join(t.TempDir(), "engine.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, engineStore.Close()) })

	runners := runner.NewRegistry()
	require.NoError(t, runners.Register(runner.NewJSRunner(registry)))

	s, err := scheduler.New(engineStore, runners, scheduler.Config{
		MaxWorkers:           8,
		PollInterval:         25 * time.Millisecond,
		DefaultLeaseDuration: 30 * time.Second,
	}, "worker-js-demo", nil)
	require.NoError(t, err)
	s.SetSiteDBProvider(func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error) {
		require.Equal(t, model.SiteName("js-demo"), site)
		return siteDB, nil
	})

	params, summaryOpID, err := BuildWorkflow(RunOptions{
		WorkflowID: "wf-js-demo",
		Count:      3,
		Multiplier: 5,
		Prefix:     "spec",
	})
	require.NoError(t, err)
	require.NoError(t, s.CreateWorkflow(ctx, params))

	for i := 0; i < 8; i++ {
		_, err = s.RunOnce(ctx)
		require.NoError(t, err)

		workflow, err := engineStore.GetWorkflow(ctx, params.Workflow.ID)
		require.NoError(t, err)
		require.NotNil(t, workflow)
		if workflow.Status == model.WorkflowStatusSucceeded {
			break
		}
		require.NotEqual(t, model.WorkflowStatusFailed, workflow.Status)
	}

	workflow, err := engineStore.GetWorkflow(ctx, params.Workflow.ID)
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusSucceeded, workflow.Status)

	result, err := engineStore.GetResult(ctx, params.Workflow.ID, summaryOpID)
	require.NoError(t, err)
	require.NotNil(t, result)

	var summary struct {
		RunID         string   `json:"runID"`
		ItemCount     int      `json:"itemCount"`
		TotalBase     int      `json:"totalBase"`
		TotalSquared  int      `json:"totalSquared"`
		Labels        []string `json:"labels"`
		ArtifactNames []string `json:"artifactNames"`
	}
	require.NoError(t, json.Unmarshal(result.Data, &summary))
	require.Equal(t, "wf-js-demo", summary.RunID)
	require.Equal(t, 3, summary.ItemCount)
	require.Equal(t, 30, summary.TotalBase)
	require.Equal(t, 350, summary.TotalSquared)
	require.Len(t, summary.Labels, 3)
	require.Len(t, summary.ArtifactNames, 3)

	row := siteDB.QueryRow(`
		SELECT item_count, total_base, total_squared, labels_json, artifact_names_json
		FROM demo_runs
		WHERE run_id = ?
	`, "wf-js-demo")

	var itemCount int
	var totalBase int
	var totalSquared int
	var labelsJSON string
	var artifactNamesJSON string
	require.NoError(t, row.Scan(&itemCount, &totalBase, &totalSquared, &labelsJSON, &artifactNamesJSON))
	require.Equal(t, 3, itemCount)
	require.Equal(t, 30, totalBase)
	require.Equal(t, 350, totalSquared)
	require.Contains(t, labelsJSON, "SPEC item 1")
	require.Contains(t, artifactNamesJSON, "spec-01.json")
}

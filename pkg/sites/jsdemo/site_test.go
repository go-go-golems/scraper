package jsdemo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	sitemigrate "github.com/go-go-golems/scraper/pkg/sites/migrate"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestDefinitionLoadsEmbeddedManifest(t *testing.T) {
	def := Definition()

	require.Equal(t, model.SiteName("js-demo"), def.Name)
	require.Equal(t, "js-demo.db", def.DatabaseFileName)
	require.Equal(t, "scripts", def.ScriptsRoot)
	require.Equal(t, "verbs", def.VerbsRoot)
	require.Equal(t, "migrations", def.SQLMigrationsRoot)
	require.Len(t, def.Modules, 1)
}

func TestJSDemoSeedWorkflow(t *testing.T) {
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

	params, summaryOpID, err := BuildSeedWorkflow(RunOptions{
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

func TestJSDemoItemWorkflow(t *testing.T) {
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
	}, "worker-js-demo-item", nil)
	require.NoError(t, err)
	s.SetSiteDBProvider(func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error) {
		require.Equal(t, model.SiteName("js-demo"), site)
		return siteDB, nil
	})

	params, itemOpID, err := BuildItemWorkflow(RunOptions{
		WorkflowID: "wf-js-demo-item",
		Index:      2,
		Multiplier: 4,
		Prefix:     "solo",
	})
	require.NoError(t, err)
	require.NoError(t, s.CreateWorkflow(ctx, params))

	for i := 0; i < 4; i++ {
		_, err = s.RunOnce(ctx)
		require.NoError(t, err)
	}

	workflow, err := engineStore.GetWorkflow(ctx, params.Workflow.ID)
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusSucceeded, workflow.Status)

	result, err := engineStore.GetResult(ctx, params.Workflow.ID, itemOpID)
	require.NoError(t, err)
	require.NotNil(t, result)

	var item struct {
		RunID        string `json:"runID"`
		ItemKey      string `json:"itemKey"`
		Index        int    `json:"index"`
		BaseValue    int    `json:"baseValue"`
		SquaredValue int    `json:"squaredValue"`
		Label        string `json:"label"`
	}
	require.NoError(t, json.Unmarshal(result.Data, &item))
	require.Equal(t, "wf-js-demo-item", item.RunID)
	require.Equal(t, "solo-03", item.ItemKey)
	require.Equal(t, 2, item.Index)
	require.Equal(t, 12, item.BaseValue)
	require.Equal(t, 144, item.SquaredValue)
	require.Equal(t, "SOLO item 3", item.Label)
}

func TestJSDemoJSQueueProcessesOneOpPerCycle(t *testing.T) {
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
	}, "worker-js-demo-queue", nil)
	require.NoError(t, err)
	s.SetSiteDBProvider(func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error) {
		require.Equal(t, model.SiteName("js-demo"), site)
		return siteDB, nil
	})

	ops := make([]model.OpSpec, 0, 2)
	for i := 0; i < 2; i++ {
		input, err := itemInput(RunOptions{
			WorkflowID: "wf-js-demo-queue",
			Index:      i,
			Multiplier: 7,
			Prefix:     "queue",
		})
		require.NoError(t, err)

		ops = append(ops, model.OpSpec{
			ID:         itemOpID("wf-js-demo-queue", i),
			WorkflowID: "wf-js-demo-queue",
			Site:       model.SiteName("js-demo"),
			Kind:       "js",
			Queue:      model.QueueKey("site:js-demo:js"),
			DedupKey:   fmt.Sprintf("js-demo:queue:%02d", i+1),
			Input:      input,
			Metadata:   map[string]string{"script": "build_item.js"},
		})
	}

	require.NoError(t, s.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:   "wf-js-demo-queue",
			Site: model.SiteName("js-demo"),
			Name: "js-demo queue serialization",
		},
		Initial: ops,
	}))

	first, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, first.Processed)

	stats, err := engineStore.GetWorkflowStats(ctx, "wf-js-demo-queue")
	require.NoError(t, err)
	require.Equal(t, 2, stats.Total)
	require.Equal(t, 1, stats.Succeeded)
	require.Equal(t, 1, stats.Ready)

	second, err := s.RunOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, second.Processed)

	stats, err = engineStore.GetWorkflowStats(ctx, "wf-js-demo-queue")
	require.NoError(t, err)
	require.Equal(t, 2, stats.Succeeded)

	workflow, err := engineStore.GetWorkflow(ctx, "wf-js-demo-queue")
	require.NoError(t, err)
	require.Equal(t, model.WorkflowStatusSucceeded, workflow.Status)
}

package cmd

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/stretchr/testify/require"
)

func TestEngineStatusMissingDB(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"engine", "status", "--engine-db", filepath.Join(t.TempDir(), "missing.db")})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Exists: no")
	require.Contains(t, stdout.String(), "Current schema version: n/a")
}

func TestEngineStatusPopulatedDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "engine.db")
	populateEngineDBForCommandTest(t, dbPath)

	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"engine", "status", "--engine-db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Exists: yes")
	require.Contains(t, stdout.String(), "Migrations up to date: yes")
	require.Contains(t, stdout.String(), "Workflows: 1")
	require.Contains(t, stdout.String(), "succeeded: 1")
}

func TestEngineMigrationsStatusCommand(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "engine.db")
	populateEngineDBForCommandTest(t, dbPath)

	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"engine", "migrations", "status", "--engine-db", dbPath})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "[applied] 1 001_engine_core.sql")
	require.Contains(t, stdout.String(), "[applied] 2 002_engine_runtime.sql")
}

func populateEngineDBForCommandTest(t *testing.T, dbPath string) {
	t.Helper()

	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, dbPath)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	workflow := model.WorkflowRun{
		ID:        model.WorkflowID("wf-1"),
		Site:      model.SiteName("nereval"),
		Name:      "CLI visibility test",
		Status:    model.WorkflowStatusRunning,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: workflow,
		Initial: []model.OpSpec{
			{
				ID:         model.OpID("op-1"),
				WorkflowID: workflow.ID,
				Site:       workflow.Site,
				Kind:       "http/fetch",
				Queue:      model.QueueKey("site:nereval:http"),
				Retry: model.RetryPolicy{
					MaxAttempts:    3,
					BackoffKind:    model.BackoffKindFixed,
					InitialBackoff: time.Second,
					MaxBackoff:     time.Second,
					Multiplier:     1,
				},
			},
		},
	}))

	leasedOp, lease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         model.QueueKey("site:nereval:http"),
		Site:          workflow.Site,
		LeaseDuration: 30 * time.Second,
		Now:           time.Now().UTC(),
	})
	require.NoError(t, err)
	require.NotNil(t, leasedOp)
	require.NotNil(t, lease)

	require.NoError(t, store.CompleteOp(ctx, leasedOp.ID, storecontract.Completion{
		Lease: *lease,
		Result: model.OpResult{
			OpID:        leasedOp.ID,
			Data:        []byte(`{"ok":true}`),
			CompletedAt: time.Now().UTC(),
		},
	}))
}

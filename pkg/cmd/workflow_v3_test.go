package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/taskpackages/cookbooklinear"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3product"
	"github.com/stretchr/testify/require"
)

type workflowCLIFixture struct {
	root, database, artifacts, script, inputs string
}

func newWorkflowCLIFixture(t *testing.T) workflowCLIFixture {
	t.Helper()
	root := t.TempDir()
	script := filepath.Join(root, "workflow.js")
	customers := filepath.Join(root, "customers.jsonl")
	inputs := filepath.Join(root, "inputs.json")
	require.NoError(t, os.WriteFile(script, []byte(cookbooklinear.WorkflowSource()), 0o600))
	require.NoError(t, os.WriteFile(customers, []byte("{\"id\":\"1\",\"email\":\"A@EXAMPLE.COM\"}\n"), 0o600))
	require.NoError(t, os.WriteFile(inputs, []byte(`{"source":{"path":"customers.jsonl","schema":"customer-jsonl-ref/v1","mediaType":"application/x-ndjson"}}`), 0o600))
	return workflowCLIFixture{
		root: root, database: filepath.Join(root, "workflow.db"),
		artifacts: filepath.Join(root, "artifacts"), script: script, inputs: inputs,
	}
}

func (f workflowCLIFixture) flags() []string {
	return []string{"--workflow-db", f.database, "--artifact-root", f.artifacts, "--poll-interval", "5ms"}
}

func executeScraper(t *testing.T, ctx context.Context, args ...string) string {
	t.Helper()
	root, err := NewRootCommand("test-version")
	require.NoError(t, err)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs(args)
	root.SetContext(ctx)
	require.NoError(t, root.Execute(), output.String())
	return output.String()
}

func TestWorkflowV3CLIValidateCompileRunInspectAndCancel(t *testing.T) {
	fixture := newWorkflowCLIFixture(t)
	ctx := context.Background()
	validation := executeScraper(t, ctx, append([]string{"workflow"}, append(fixture.flags(), "validate", fixture.script)...)...)
	require.Contains(t, validation, `"ok": true`)

	planPath := filepath.Join(fixture.root, "plan.json")
	executeScraper(t, ctx, append([]string{"workflow"}, append(fixture.flags(), "compile", fixture.script, "--out", planPath)...)...)
	planBody, err := os.ReadFile(planPath)
	require.NoError(t, err)
	var plan workflowv3.WorkflowPlan
	require.NoError(t, json.Unmarshal(planBody, &plan))
	require.NotEmpty(t, plan.Digest)

	runOutput := executeScraper(t, ctx, append([]string{"workflow"}, append(fixture.flags(), "run", fixture.script, "--inputs", fixture.inputs, "--run-id", "cli-run")...)...)
	require.Contains(t, runOutput, `"status": "succeeded"`)
	listOutput := executeScraper(t, ctx, append([]string{"workflow"}, append(fixture.flags(), "runs", "list")...)...)
	require.Contains(t, listOutput, `"runId": "cli-run"`)
	showOutput := executeScraper(t, ctx, append([]string{"workflow"}, append(fixture.flags(), "runs", "show", "cli-run")...)...)
	require.Contains(t, showOutput, `"planDigest"`)
	followOutput := executeScraper(t, ctx, append([]string{"workflow"}, append(fixture.flags(), "runs", "follow", "cli-run")...)...)
	require.Contains(t, followOutput, `"status":"succeeded"`)

	executeScraper(t, ctx, append([]string{"workflow"}, append(fixture.flags(), "submit", fixture.script, "--inputs", fixture.inputs, "--run-id", "cancel-run")...)...)
	cancelOutput := executeScraper(t, ctx, append([]string{"workflow"}, append(fixture.flags(), "runs", "cancel", "cancel-run")...)...)
	require.Contains(t, cancelOutput, `"status": "canceled"`)

	packages := executeScraper(t, ctx, "task-packages", "list")
	require.Contains(t, packages, `"name": "cookbook-linear"`)
}

func TestWorkflowV3CLIRunReturnsFailureAfterWritingTerminalEvidence(t *testing.T) {
	fixture := newWorkflowCLIFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(fixture.root, "customers.jsonl"), []byte("{\"id\":\"1\",\"email\":\"a@example.com\"}\n{\"id\":\"1\",\"email\":\"b@example.com\"}\n"), 0o600))
	root, err := NewRootCommand("test-version")
	require.NoError(t, err)
	var output bytes.Buffer
	root.SetOut(&output)
	root.SetErr(&output)
	root.SetArgs(append([]string{"workflow"}, append(fixture.flags(), "run", fixture.script, "--inputs", fixture.inputs, "--run-id", "cli-failure")...))
	err = root.Execute()
	require.ErrorContains(t, err, "finished with status failed")
	require.Contains(t, output.String(), `"status": "failed"`)
	require.Contains(t, output.String(), "CUSTOMER_DUPLICATE_ID")
}

func TestWorkflowV3CLIWorkerRecoversSubmittedRunAfterRestart(t *testing.T) {
	fixture := newWorkflowCLIFixture(t)
	ctx := context.Background()
	executeScraper(t, ctx, append([]string{"workflow"}, append(fixture.flags(), "submit", fixture.script, "--inputs", fixture.inputs, "--run-id", "worker-restart")...)...)

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	workerDone := make(chan error, 1)
	go func() {
		root, err := NewRootCommand("test-version")
		if err != nil {
			workerDone <- err
			return
		}
		root.SetContext(workerCtx)
		root.SetArgs(append([]string{"worker"}, append(fixture.flags(), "run")...))
		workerDone <- root.Execute()
	}()

	config := workflowv3product.DefaultConfig()
	config.DatabasePath, config.ArtifactRoot = fixture.database, fixture.artifacts
	config.PollInterval = 5 * time.Millisecond
	app, err := workflowv3product.Open(ctx, config)
	require.NoError(t, err)
	deadline := time.Now().Add(5 * time.Second)
	for {
		view, snapshotErr := app.Show(ctx, "worker-restart")
		require.NoError(t, snapshotErr)
		if view.Snapshot.Status == "succeeded" {
			break
		}
		require.True(t, time.Now().Before(deadline), "worker did not complete submitted run")
		time.Sleep(10 * time.Millisecond)
	}
	require.NoError(t, app.Close())
	cancelWorker()
	require.NoError(t, <-workerDone)
}

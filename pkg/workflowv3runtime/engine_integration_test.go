package workflowv3runtime

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3linear"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/require"
)

func TestEngineRunsAuthoredWorkflowAcrossRestartWithoutPersistingSource(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	databasePath := filepath.Join(root, "workflow.db")
	artifactRoot := filepath.Join(root, "artifacts")
	privateCanary := "PRIVATE-SOURCE-CANARY-7e6ec43c"
	secretToken := "TOP-SECRET-TOKEN-4f73d312"

	registry, err := workflowv3linear.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	authored, err := workflowmodule.Author(
		ctx,
		workflowv3linear.WorkflowSource(),
		catalog,
		workflowv3linear.DescriptorModule(),
	)
	require.NoError(t, err)

	artifacts, err := workflowv3.NewFileArtifactStore(artifactRoot, 8<<20)
	require.NoError(t, err)
	sourceBody := largeCustomerSource(privateCanary, secretToken, 12000)
	sourceRef, err := artifacts.Put(
		ctx,
		"customer-jsonl-ref/v1",
		"application/x-ndjson",
		sourceBody,
	)
	require.NoError(t, err)

	now := time.Date(2026, 7, 21, 18, 0, 0, 0, time.UTC)
	store, err := workflowv3sqlite.Open(ctx, databasePath)
	require.NoError(t, err)
	engine := &Engine{
		Store: store, Registry: registry, Artifacts: artifacts,
		Modules:       fsTaskModules(t),
		LeaseDuration: time.Minute, Now: func() time.Time { return now },
	}
	require.NoError(t, engine.Submit(ctx, "linear-real-1", authored.Plan, map[string]workflowv3.ArtifactRef{
		"source": sourceRef,
	}))
	ran, err := engine.RunOne(ctx)
	require.NoError(t, err)
	require.True(t, ran)
	beforeRestart, err := engine.Snapshot(ctx, "linear-real-1")
	require.NoError(t, err)
	require.Equal(t, "running", beforeRestart.Status)
	require.Len(t, beforeRestart.Attempts, 1)
	require.NoError(t, store.Close())

	reopenedStore, err := workflowv3sqlite.Open(ctx, databasePath)
	require.NoError(t, err)
	restarted := &Engine{
		Store: reopenedStore, Registry: registry, Artifacts: artifacts,
		Modules:       fsTaskModules(t),
		LeaseDuration: time.Minute, Now: func() time.Time { return now.Add(time.Second) },
	}
	require.NoError(t, restarted.RunUntilIdle(ctx))
	completed, err := restarted.Snapshot(ctx, "linear-real-1")
	require.NoError(t, err)
	require.Equal(t, "succeeded", completed.Status)
	require.Len(t, completed.Attempts, 2)
	require.Equal(t, registry.Generation(), completed.Attempts[0].RegistryGeneration)
	outputRef := completed.Outputs["dataset"]
	require.Equal(t, "validated-customers-ref/v1", outputRef.Schema)
	outputBody, err := workflowv3.ReadArtifact(ctx, artifacts, outputRef)
	require.NoError(t, err)
	require.Contains(t, string(outputBody), `"count":12000`)
	require.NotContains(t, string(outputBody), privateCanary)
	require.NotContains(t, string(outputBody), secretToken)
	require.NoError(t, reopenedStore.Close())

	finalStore, err := workflowv3sqlite.Open(ctx, databasePath)
	require.NoError(t, err)
	reopenedSnapshot, err := finalStore.Snapshot(ctx, "linear-real-1")
	require.NoError(t, err)
	require.Equal(t, completed.Outputs, reopenedSnapshot.Outputs)
	require.Equal(t, completed.PlanDigest, reopenedSnapshot.PlanDigest)
	require.NoError(t, finalStore.Close())

	persisted, persistedBytes := readSQLiteFiles(t, databasePath)
	require.NotContains(t, string(persisted), privateCanary)
	require.NotContains(t, string(persisted), secretToken)
	require.Less(t, persistedBytes, int64(len(sourceBody))/2)
	t.Logf(
		"privacy/storage evidence: source=%d persistedSQLite=%d ratio=%.4f",
		len(sourceBody),
		persistedBytes,
		float64(persistedBytes)/float64(len(sourceBody)),
	)

	sourceArtifact, err := workflowv3.ReadArtifact(ctx, artifacts, sourceRef)
	require.NoError(t, err)
	require.Contains(t, string(sourceArtifact), privateCanary)
	require.Contains(t, string(sourceArtifact), secretToken)
}

func TestEnginePersistsTypedTaskFailureWithoutTaskMessage(t *testing.T) {
	ctx := context.Background()
	registry, err := workflowv3linear.Registry()
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	authored, err := workflowmodule.Author(
		ctx,
		workflowv3linear.WorkflowSource(),
		catalog,
		workflowv3linear.DescriptorModule(),
	)
	require.NoError(t, err)
	artifacts, err := workflowv3.NewFileArtifactStore(t.TempDir(), 1<<20)
	require.NoError(t, err)
	source, err := artifacts.Put(
		ctx,
		"customer-jsonl-ref/v1",
		"application/x-ndjson",
		[]byte("{\"id\":\"1\",\"email\":\"a@example.com\"}\n"+
			"{\"id\":\"1\",\"email\":\"b@example.com\"}\n"),
	)
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(ctx, filepath.Join(t.TempDir(), "failure.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	engine := &Engine{
		Store: store, Registry: registry, Artifacts: artifacts,
		Modules: fsTaskModules(t),
	}
	require.NoError(t, engine.Submit(ctx, "duplicate", authored.Plan, map[string]workflowv3.ArtifactRef{
		"source": source,
	}))
	err = engine.RunUntilIdle(ctx)
	require.ErrorContains(t, err, "CUSTOMER_DUPLICATE_ID")
	snapshot, err := engine.Snapshot(ctx, "duplicate")
	require.NoError(t, err)
	require.Equal(t, "failed", snapshot.Status)
	require.Len(t, snapshot.Attempts, 2)
	failure := snapshot.Attempts[1].Failure
	require.Equal(t, "validation", failure.Class)
	require.Equal(t, "CUSTOMER_DUPLICATE_ID", failure.Code)
	require.Equal(t, "task reported CUSTOMER_DUPLICATE_ID", failure.Message)
}

func largeCustomerSource(canary, secret string, count int) []byte {
	var buffer bytes.Buffer
	for i := 0; i < count; i++ {
		fmt.Fprintf(
			&buffer,
			"{\"id\":\" %06d \",\"email\":\"USER%06d@EXAMPLE.COM \","+
				"\"private\":\"%s-%06d\",\"token\":\"%s\"}\n",
			i,
			i,
			canary,
			i,
			secret,
		)
	}
	return buffer.Bytes()
}

func readSQLiteFiles(t *testing.T, databasePath string) ([]byte, int64) {
	t.Helper()
	var combined []byte
	var total int64
	for _, path := range []string{databasePath, databasePath + "-wal", databasePath + "-shm"} {
		body, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		require.NoError(t, err)
		combined = append(combined, body...)
		total += int64(len(body))
	}
	return combined, total
}

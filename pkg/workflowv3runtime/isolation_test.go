package workflowv3runtime

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3isolation"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func TestBoundedProtocolFrameRejectsMalformedAndOversizedInput(t *testing.T) {
	body, err := readBoundedFrame(strings.NewReader("{\"ok\":true}\n"), 64)
	require.NoError(t, err)
	require.JSONEq(t, `{"ok":true}`, string(body))
	for _, input := range []string{"", "{}", "{}\n{}\n", " {}\n", "{} \n"} {
		_, err := readBoundedFrame(strings.NewReader(input), 64)
		require.Error(t, err, input)
	}
	_, err = readBoundedFrame(strings.NewReader(strings.Repeat("x", 65)+"\n"), 64)
	require.ErrorContains(t, err, "exceeds")
}

func FuzzReadBoundedIsolationFrame(f *testing.F) {
	f.Add([]byte("{}\n"))
	f.Add([]byte("{}"))
	f.Fuzz(func(t *testing.T, body []byte) {
		result, err := readBoundedFrame(bytes.NewReader(body), 1024)
		if err == nil {
			require.LessOrEqual(t, len(result), 1024)
			require.NotContains(t, string(result), "\n")
		}
	})
}

func TestCandidateValidationRejectsTraversalSymlinkHardlinkAndDigestDrift(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "objects"), 0o700))
	body := []byte("candidate")
	store, err := workflowv3.NewFileArtifactStore(root, 1024)
	require.NoError(t, err)
	ref, err := store.Put(context.Background(), "output/v1", "text/plain", body)
	require.NoError(t, err)
	read, err := readRegularCandidate(root, ref)
	require.NoError(t, err)
	require.Equal(t, body, read)

	traversal := ref
	traversal.Locator = "../outside"
	_, err = readRegularCandidate(root, traversal)
	require.Error(t, err)

	symlink := ref
	symlink.Locator = "objects/symlink"
	require.NoError(t, os.Symlink(filepath.Join(root, filepath.FromSlash(ref.Locator)), filepath.Join(root, "objects", "symlink")))
	_, err = readRegularCandidate(root, symlink)
	require.ErrorContains(t, err, "regular")

	hardlink := ref
	hardlink.Locator = "objects/hardlink"
	require.NoError(t, os.Link(filepath.Join(root, filepath.FromSlash(ref.Locator)), filepath.Join(root, "objects", "hardlink")))
	_, err = readRegularCandidate(root, hardlink)
	require.ErrorContains(t, err, "hard links")
	require.NoError(t, os.Remove(filepath.Join(root, "objects", "hardlink")))

	drift := ref
	drift.Digest = "sha256:" + strings.Repeat("0", 64)
	_, err = readRegularCandidate(root, drift)
	require.ErrorContains(t, err, "digest")

	destination, err := workflowv3.NewFileArtifactStore(t.TempDir(), 1024)
	require.NoError(t, err)
	policy := workflowv3isolation.Policy()
	policy.MaxOutputBytes = 4
	_, err = publishCandidates(context.Background(), root, destination,
		map[string]string{"output": "output/v1"}, map[string]workflowv3.ArtifactRef{"output": ref}, policy)
	require.ErrorContains(t, err, "exceeds compiled limits")
	policy.MaxOutputBytes = 1024
	policy.MaxOutputFiles = 0
	_, err = publishCandidates(context.Background(), root, destination,
		map[string]string{"output": "output/v1"}, map[string]workflowv3.ArtifactRef{"output": ref}, policy)
	require.ErrorContains(t, err, "exceeds compiled limits")
}

func TestRestrictedWorkerWallTimeAndCancellationKillChild(t *testing.T) {
	ctx := context.Background()
	worker := buildIsolatedWorker(t)
	executor := &BubblewrapExecutor{WorkerExecutable: worker, BubblewrapExecutable: "/usr/bin/bwrap"}
	executorDigest, err := executor.Identity()
	require.NoError(t, err)
	registry, err := workflowv3isolation.RegistryWithExecutor(executorDigest)
	require.NoError(t, err)
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	spec, ok := catalog.Lookup(workflowv3.TaskKey{Kind: "fixture.isolation.spin", Version: "v1"})
	require.True(t, ok)
	registered, err := registry.Resolve(spec.Identity)
	require.NoError(t, err)
	artifacts, err := workflowv3.NewFileArtifactStore(filepath.Join(t.TempDir(), "artifacts"), 1024)
	require.NoError(t, err)
	input, err := artifacts.Put(ctx, "isolation-source/v1", "application/json", []byte(`{"values":[]}`))
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(FSInputModule())
	require.NoError(t, err)
	request := TaskRequest{RunID: "limits", NodeKey: "spin", Attempt: 1, Task: registered, Inputs: map[string]workflowv3.ArtifactRef{"source": input}, Artifacts: artifacts, Modules: modules}

	cpuPolicy := workflowv3isolation.Policy()
	cpuPolicy.CPUTimeMillis = 100
	cpuIsolation, err := workflowv3.CompileIsolation(&cpuPolicy, spec.IsolationMaximum, spec.IsolationExecutorDigest)
	require.NoError(t, err)
	_, err = executor.Execute(ctx, request, cpuIsolation)
	var failure *TaskFailureError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "resource", failure.Failure.Class)
	require.Equal(t, "ISOLATION_CPU_LIMIT", failure.Failure.Code)

	requested := workflowv3isolation.Policy()
	requested.WallTimeMillis = 10
	isolation, err := workflowv3.CompileIsolation(&requested, spec.IsolationMaximum, spec.IsolationExecutorDigest)
	require.NoError(t, err)
	_, err = executor.Execute(ctx, request, isolation)
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "resource", failure.Failure.Class)
	require.Equal(t, "ISOLATION_WALL_TIME", failure.Failure.Code)

	isolation, err = workflowv3.CompileIsolation(nil, spec.IsolationMaximum, spec.IsolationExecutorDigest)
	require.NoError(t, err)
	cancelCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	_, err = executor.Execute(cancelCtx, request, isolation)
	require.ErrorAs(t, err, &failure)
	require.Equal(t, "canceled", failure.Failure.Class)
	require.Equal(t, "ISOLATION_CANCELED", failure.Failure.Code)
	require.True(t, errors.Is(cancelCtx.Err(), context.DeadlineExceeded))
}

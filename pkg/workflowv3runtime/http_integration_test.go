package workflowv3runtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	fetchmod "github.com/go-go-golems/go-go-goja/modules/fetch"
	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/testfixtures/workflowv3http"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
	"github.com/stretchr/testify/require"
)

func TestHTTPSnapshotRetriesAndReopensWithoutPersistingRequestSecrets(t *testing.T) {
	var requests atomic.Int32
	responseCanary := "PUBLIC-ARTICLE-CANARY-73db"
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(response, "temporary", http.StatusServiceUnavailable)
			return
		}
		_, _ = response.Write([]byte(responseCanary))
	}))
	defer server.Close()

	root := t.TempDir()
	databasePath := filepath.Join(root, "workflow.db")
	secret := "HTTP-QUERY-SECRET-81ca"
	engine, dispatcher, artifacts := newHTTPEngine(
		t,
		databasePath,
		filepath.Join(root, "artifacts"),
		[]string{server.URL},
		128<<10,
	)
	input := putJSONArtifact(t, artifacts, "http-article-list-ref/v1", map[string]any{
		"urls": []string{server.URL + "/article?token=" + secret},
	})
	authored := authoredHTTPPlan(t, engine.Registry)
	require.NoError(t, engine.Submit(context.Background(), "http-retry", authored.Plan, map[string]workflowv3.ArtifactRef{
		"articles": input,
	}))

	snapshot := runDispatcherUntilStatus(t, dispatcher, engine, "http-retry", "succeeded")
	require.Len(t, snapshot.Attempts, 2)
	require.Equal(t, "failed", snapshot.Attempts[0].Status)
	require.True(t, snapshot.Attempts[0].Failure.Retryable)
	require.Equal(t, "HTTP_FETCH_SERVER", snapshot.Attempts[0].Failure.Code)
	require.Equal(t, workflowv3http.ResourceClass, snapshot.Attempts[0].ResourceClass)
	require.Equal(t, "succeeded", snapshot.Attempts[1].Status)
	require.Equal(t, int32(2), requests.Load())

	body, err := workflowv3.ReadArtifact(context.Background(), artifacts, snapshot.Outputs["snapshot"])
	require.NoError(t, err)
	require.Contains(t, string(body), responseCanary)
	require.Contains(t, string(body), `"count":1`)
	require.NotContains(t, string(body), secret)
	require.NoError(t, engine.Store.Close())

	reopened, err := workflowv3sqlite.Open(context.Background(), databasePath)
	require.NoError(t, err)
	reopenedSnapshot, err := reopened.Snapshot(context.Background(), "http-retry")
	require.NoError(t, err)
	require.Equal(t, snapshot.Outputs, reopenedSnapshot.Outputs)
	require.NoError(t, reopened.Close())

	persisted, _ := readSQLiteFiles(t, databasePath)
	require.NotContains(t, string(persisted), secret)
	require.NotContains(t, string(persisted), responseCanary)
}

func TestHTTPSnapshotDeniesUnadvertisedOriginAndRedactsURL(t *testing.T) {
	allowed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer allowed.Close()
	denied := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("denied origin was contacted")
	}))
	defer denied.Close()

	root := t.TempDir()
	databasePath := filepath.Join(root, "workflow.db")
	engine, dispatcher, artifacts := newHTTPEngine(
		t,
		databasePath,
		filepath.Join(root, "artifacts"),
		[]string{allowed.URL},
		1024,
	)
	secret := "DENIED-URL-SECRET-22df"
	input := putJSONArtifact(t, artifacts, "http-article-list-ref/v1", map[string]any{
		"urls": []string{denied.URL + "/private?credential=" + secret},
	})
	authored := authoredHTTPPlan(t, engine.Registry)
	require.NoError(t, engine.Submit(context.Background(), "http-denied", authored.Plan, map[string]workflowv3.ArtifactRef{
		"articles": input,
	}))
	snapshot := runDispatcherUntilStatus(t, dispatcher, engine, "http-denied", "failed")
	require.Len(t, snapshot.Attempts, 3)
	for _, attempt := range snapshot.Attempts {
		require.Equal(t, "HTTP_FETCH_TRANSPORT", attempt.Failure.Code)
		require.Equal(t, "task reported HTTP_FETCH_TRANSPORT", attempt.Failure.Message)
	}
	require.NoError(t, engine.Store.Close())
	persisted, _ := readSQLiteFiles(t, databasePath)
	require.NotContains(t, string(persisted), secret)
	require.NotContains(t, string(persisted), denied.URL)
}

func TestHTTPSnapshotRejectsURLCredentials(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		requests.Add(1)
	}))
	defer server.Close()
	root := t.TempDir()
	databasePath := filepath.Join(root, "workflow.db")
	engine, dispatcher, artifacts := newHTTPEngine(
		t, databasePath, filepath.Join(root, "artifacts"),
		[]string{server.URL}, 1024,
	)
	secret := "URL-PASSWORD-SECRET-16ab"
	credentialURL := strings.Replace(server.URL, "://", "://public:"+secret+"@", 1)
	input := putJSONArtifact(t, artifacts, "http-article-list-ref/v1", map[string]any{
		"urls": []string{credentialURL},
	})
	authored := authoredHTTPPlan(t, engine.Registry)
	require.NoError(t, engine.Submit(context.Background(), "http-credentials", authored.Plan, map[string]workflowv3.ArtifactRef{
		"articles": input,
	}))
	snapshot := runDispatcherUntilStatus(t, dispatcher, engine, "http-credentials", "failed")
	require.Len(t, snapshot.Attempts, 3)
	require.Zero(t, requests.Load())
	require.NoError(t, engine.Store.Close())
	persisted, _ := readSQLiteFiles(t, databasePath)
	require.NotContains(t, string(persisted), secret)
}

func TestHTTPSnapshotClassifiesRateLimitAndTerminalStatus(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		code      string
		class     string
		retryable bool
		attempts  int
	}{
		{
			name: "rate limit", status: http.StatusTooManyRequests,
			code: "HTTP_FETCH_RATE_LIMIT", class: "rate-limit",
			retryable: true, attempts: 3,
		},
		{
			name: "not found", status: http.StatusNotFound,
			code: "HTTP_FETCH_STATUS", class: "validation",
			retryable: false, attempts: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(test.status)
			}))
			defer server.Close()
			root := t.TempDir()
			engine, dispatcher, artifacts := newHTTPEngine(
				t,
				filepath.Join(root, "workflow.db"),
				filepath.Join(root, "artifacts"),
				[]string{server.URL},
				1024,
			)
			input := putJSONArtifact(t, artifacts, "http-article-list-ref/v1", map[string]any{
				"urls": []string{server.URL},
			})
			authored := authoredHTTPPlan(t, engine.Registry)
			runID := workflowv3.RunID("http-status-" + strings.ReplaceAll(test.name, " ", "-"))
			require.NoError(t, engine.Submit(context.Background(), runID, authored.Plan, map[string]workflowv3.ArtifactRef{
				"articles": input,
			}))
			snapshot := runDispatcherUntilStatus(t, dispatcher, engine, runID, "failed")
			require.Len(t, snapshot.Attempts, test.attempts)
			for _, attempt := range snapshot.Attempts {
				require.Equal(t, test.code, attempt.Failure.Code)
				require.Equal(t, test.class, attempt.Failure.Class)
				require.Equal(t, test.retryable, attempt.Failure.Retryable)
			}
			require.NoError(t, engine.Store.Close())
		})
	}
}

func TestHTTPSnapshotRejectsRedirectOutsideAllowlist(t *testing.T) {
	var deniedRequests atomic.Int32
	denied := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		deniedRequests.Add(1)
	}))
	defer denied.Close()
	secret := "REDIRECT-SECRET-bc12"
	allowed := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Set("Location", denied.URL+"/private?token="+secret)
		response.WriteHeader(http.StatusFound)
	}))
	defer allowed.Close()
	root := t.TempDir()
	databasePath := filepath.Join(root, "workflow.db")
	engine, dispatcher, artifacts := newHTTPEngine(
		t, databasePath, filepath.Join(root, "artifacts"),
		[]string{allowed.URL}, 1024,
	)
	input := putJSONArtifact(t, artifacts, "http-article-list-ref/v1", map[string]any{
		"urls": []string{allowed.URL},
	})
	authored := authoredHTTPPlan(t, engine.Registry)
	require.NoError(t, engine.Submit(context.Background(), "http-redirect", authored.Plan, map[string]workflowv3.ArtifactRef{
		"articles": input,
	}))
	snapshot := runDispatcherUntilStatus(t, dispatcher, engine, "http-redirect", "failed")
	require.Equal(t, "HTTP_FETCH_TRANSPORT", snapshot.Attempts[2].Failure.Code)
	require.Zero(t, deniedRequests.Load())
	require.NoError(t, engine.Store.Close())
	persisted, _ := readSQLiteFiles(t, databasePath)
	require.NotContains(t, string(persisted), secret)
	require.NotContains(t, string(persisted), denied.URL)
}

func TestHTTPSnapshotEnforcesResponseLimitAndCancellation(t *testing.T) {
	t.Run("response limit", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			_, _ = response.Write([]byte(strings.Repeat("x", 2048)))
		}))
		defer server.Close()
		root := t.TempDir()
		engine, dispatcher, artifacts := newHTTPEngine(
			t,
			filepath.Join(root, "workflow.db"),
			filepath.Join(root, "artifacts"),
			[]string{server.URL},
			64,
		)
		input := putJSONArtifact(t, artifacts, "http-article-list-ref/v1", map[string]any{
			"urls": []string{server.URL},
		})
		authored := authoredHTTPPlan(t, engine.Registry)
		require.NoError(t, engine.Submit(context.Background(), "http-limit", authored.Plan, map[string]workflowv3.ArtifactRef{
			"articles": input,
		}))
		snapshot := runDispatcherUntilStatus(t, dispatcher, engine, "http-limit", "failed")
		require.Len(t, snapshot.Attempts, 3)
		require.Equal(t, "HTTP_FETCH_TRANSPORT", snapshot.Attempts[2].Failure.Code)
		require.NoError(t, engine.Store.Close())
	})

	t.Run("cancel in flight", func(t *testing.T) {
		started := make(chan struct{})
		canceled := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			close(started)
			<-request.Context().Done()
			close(canceled)
		}))
		defer server.Close()
		root := t.TempDir()
		engine, dispatcher, artifacts := newHTTPEngine(
			t,
			filepath.Join(root, "workflow.db"),
			filepath.Join(root, "artifacts"),
			[]string{server.URL},
			1024,
		)
		input := putJSONArtifact(t, artifacts, "http-article-list-ref/v1", map[string]any{
			"urls": []string{server.URL},
		})
		authored := authoredHTTPPlan(t, engine.Registry)
		require.NoError(t, engine.Submit(context.Background(), "http-cancel", authored.Plan, map[string]workflowv3.ArtifactRef{
			"articles": input,
		}))
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- dispatcher.Run(ctx) }()
		require.Eventually(t, func() bool {
			select {
			case <-started:
				return true
			default:
				return false
			}
		}, time.Second, 5*time.Millisecond)
		require.NoError(t, engine.Store.Cancel(context.Background(), "http-cancel", time.Now().UTC()))
		require.Eventually(t, func() bool {
			select {
			case <-canceled:
				return true
			default:
				return false
			}
		}, time.Second, 5*time.Millisecond)
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
		snapshot, err := engine.Snapshot(context.Background(), "http-cancel")
		require.NoError(t, err)
		require.Equal(t, "canceled", snapshot.Status)
		require.Equal(t, "canceled", snapshot.Attempts[0].Status)
		require.NoError(t, engine.Store.Close())
	})
}

func newHTTPEngine(
	t *testing.T,
	databasePath string,
	artifactRoot string,
	allowedOrigins []string,
	maxResponseBytes int64,
) (*Engine, *Dispatcher, workflowv3.ArtifactStore) {
	t.Helper()
	registry, err := workflowv3http.Registry()
	require.NoError(t, err)
	modules, err := NewTaskModuleRegistry(
		FSInputModule(),
		FetchModule(workflowv3http.FetchAlias, fetchmod.Policy{
			AllowedOrigins:   allowedOrigins,
			Timeout:          time.Second,
			MaxResponseBytes: maxResponseBytes,
			Credentials: fetchmod.CredentialPolicy{
				AllowEnv: false, AllowFiles: false,
			},
		}, nil),
	)
	require.NoError(t, err)
	artifacts, err := workflowv3.NewFileArtifactStore(artifactRoot, 1<<20)
	require.NoError(t, err)
	store, err := workflowv3sqlite.Open(context.Background(), databasePath)
	require.NoError(t, err)
	engine := &Engine{
		Store: store, Registry: registry, Artifacts: artifacts,
		Modules: modules, LeaseDuration: 2 * time.Second,
	}
	dispatcher := &Dispatcher{
		Engine:       engine,
		Capacities:   map[string]int{workflowv3http.ResourceClass: 1},
		PollInterval: 2 * time.Millisecond,
	}
	return engine, dispatcher, artifacts
}

func authoredHTTPPlan(t *testing.T, registry workflowv3.RegistryResolver) workflowmodule.AuthoringResult {
	t.Helper()
	catalog, err := registry.Catalog()
	require.NoError(t, err)
	authored, err := workflowmodule.Author(
		context.Background(),
		workflowv3http.WorkflowSource(),
		catalog,
		workflowv3http.DescriptorModule(),
	)
	require.NoError(t, err)
	return authored
}

func putJSONArtifact(
	t *testing.T,
	artifacts workflowv3.ArtifactStore,
	schema string,
	value any,
) workflowv3.ArtifactRef {
	t.Helper()
	body, err := json.Marshal(value)
	require.NoError(t, err)
	ref, err := artifacts.Put(context.Background(), schema, "application/json", body)
	require.NoError(t, err)
	return ref
}

func runDispatcherUntilStatus(
	t *testing.T,
	dispatcher *Dispatcher,
	engine *Engine,
	runID workflowv3.RunID,
	status string,
) workflowv3.RunSnapshot {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- dispatcher.Run(ctx) }()
	var snapshot workflowv3.RunSnapshot
	require.Eventually(t, func() bool {
		var err error
		snapshot, err = engine.Snapshot(context.Background(), runID)
		return err == nil && snapshot.Status == status
	}, 5*time.Second, 5*time.Millisecond)
	cancel()
	require.True(t, errors.Is(<-done, context.Canceled))
	return snapshot
}

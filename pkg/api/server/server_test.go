package server_test

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	apiserver "github.com/go-go-golems/scraper/pkg/api/server"
	scrapercmd "github.com/go-go-golems/scraper/pkg/cmd"
	"github.com/go-go-golems/scraper/pkg/engine/config"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	"github.com/go-go-golems/scraper/pkg/sites/defaults"
	sitemigrate "github.com/go-go-golems/scraper/pkg/sites/migrate"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// sitesRegistry returns a registry with all built-in sites loaded.
func sitesRegistry(t *testing.T) *siteregistry.Registry {
	reg, err := defaults.NewRegistry()
	if err != nil {
		t.Skipf("cannot load sites registry: %v", err)
	}
	return reg
}

func TestServerHealthAndCatalogEndpoints(t *testing.T) {
	registry := sitesRegistry(t)

	server, err := apiserver.New(apiserver.Config{
		Address:      "127.0.0.1:0",
		EngineDB:     t.TempDir() + "/engine.db",
		SitesDir:     t.TempDir(),
		ReadTimeout:  5,
		WriteTimeout: 5,
		Version:      "test-version",
	}, registry)
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler)
	defer ts.Close()

	t.Run("healthz", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/healthz")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		payload := map[string]bool{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		require.Equal(t, true, payload["ok"])
	})

	t.Run("sites", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/sites")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		payload := struct {
			Sites []struct {
				Name         string `json:"name"`
				OriginKind   string `json:"originKind"`
				ManifestPath string `json:"manifestPath"`
			} `json:"sites"`
		}{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		require.NotEmpty(t, payload.Sites)
		var jsDemo *struct {
			Name         string `json:"name"`
			OriginKind   string `json:"originKind"`
			ManifestPath string `json:"manifestPath"`
		}
		for i := range payload.Sites {
			if payload.Sites[i].Name == "js-demo" {
				jsDemo = &payload.Sites[i]
				break
			}
		}
		require.NotNil(t, jsDemo)
		require.Equal(t, "manifest", jsDemo.OriginKind)
		require.Equal(t, "site.yaml", jsDemo.ManifestPath)
	})

	t.Run("site detail provenance", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/sites/js-demo/detail")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		payload := struct {
			Site struct {
				Name         string `json:"name"`
				OriginKind   string `json:"originKind"`
				ManifestPath string `json:"manifestPath"`
			} `json:"site"`
		}{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		require.Equal(t, "js-demo", payload.Site.Name)
		require.Equal(t, "manifest", payload.Site.OriginKind)
		require.Equal(t, "site.yaml", payload.Site.ManifestPath)
	})

	t.Run("metrics", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Contains(t, string(body), "scraper_http_requests_total")
		require.Contains(t, string(body), "scraper_engine_workflows_total")
	})

	t.Run("verb details", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/sites/js-demo/verbs/seed")
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		payload := struct {
			Verb struct {
				Name        string `json:"name"`
				CommandPath string `json:"commandPath"`
				SourceFile  string `json:"sourceFile"`
			} `json:"verb"`
		}{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		require.Equal(t, "seed", payload.Verb.Name)
		require.Equal(t, "site js-demo run seed", payload.Verb.CommandPath)
		require.Equal(t, "seed.js", payload.Verb.SourceFile)
	})
}

func TestServerSubmitThenWorkerAndInspectWorkflow(t *testing.T) {
	registry := sitesRegistry(t)

	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	server, err := apiserver.New(apiserver.Config{
		Address:      "127.0.0.1:0",
		EngineDB:     engineDB,
		SitesDir:     sitesDir,
		ReadTimeout:  5,
		WriteTimeout: 5,
		Version:      "test-version",
	}, registry)
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler)
	defer ts.Close()

	body := bytes.NewBufferString(`{
		"workflowID": "api-js-demo",
		"values": {
			"count": 3,
			"multiplier": 4,
			"prefix": "api"
		}
	}`)
	resp, err := http.Post(ts.URL+"/api/v1/sites/js-demo/verbs/seed:submit", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	submitPayload := struct {
		Workflow struct {
			ID string `json:"id"`
		} `json:"workflow"`
		SubmittedCount int    `json:"submittedCount"`
		TargetOpID     string `json:"targetOpID"`
	}{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&submitPayload))
	require.Equal(t, "api-js-demo", submitPayload.Workflow.ID)
	require.Equal(t, 1, submitPayload.SubmittedCount)
	require.Equal(t, "api-js-demo:seed:summary", submitPayload.TargetOpID)

	workflowResp, err := http.Get(ts.URL + "/api/v1/workflows/api-js-demo")
	require.NoError(t, err)
	defer workflowResp.Body.Close()
	require.Equal(t, http.StatusOK, workflowResp.StatusCode)

	rootCmd, err := scrapercmd.NewRootCommandWithRegistry("test-version", registry)
	require.NoError(t, err)
	var workerStdout bytes.Buffer
	rootCmd.SetOut(&workerStdout)
	rootCmd.SetErr(&workerStdout)
	rootCmd.SetArgs([]string{
		"worker", "run",
		"--sites-dir", sitesDir,
		"--engine-db", engineDB,
		"--max-cycles", "16",
		"--poll-interval", "5ms",
	})
	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, workerStdout.String(), "Succeeded:")

	statusResp, err := http.Get(ts.URL + "/api/v1/engine/status")
	require.NoError(t, err)
	defer statusResp.Body.Close()
	require.Equal(t, http.StatusOK, statusResp.StatusCode)
	statusPayload := struct {
		Status struct {
			WorkflowCount int            `json:"workflowCount"`
			OpCounts      map[string]int `json:"opCounts"`
			ResultCount   int            `json:"resultCount"`
		} `json:"status"`
	}{}
	require.NoError(t, json.NewDecoder(statusResp.Body).Decode(&statusPayload))
	require.Equal(t, 1, statusPayload.Status.WorkflowCount)
	require.Equal(t, 5, statusPayload.Status.OpCounts["succeeded"])
	require.Equal(t, 5, statusPayload.Status.ResultCount)

	workflowAfterResp, err := http.Get(ts.URL + "/api/v1/workflows/api-js-demo")
	require.NoError(t, err)
	defer workflowAfterResp.Body.Close()
	require.Equal(t, http.StatusOK, workflowAfterResp.StatusCode)
	workflowAfterPayload := struct {
		Workflow struct {
			Workflow struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"workflow"`
			Stats struct {
				Succeeded int `json:"succeeded"`
			} `json:"stats"`
		} `json:"workflow"`
	}{}
	require.NoError(t, json.NewDecoder(workflowAfterResp.Body).Decode(&workflowAfterPayload))
	require.Equal(t, "api-js-demo", workflowAfterPayload.Workflow.Workflow.ID)
	require.Equal(t, "succeeded", workflowAfterPayload.Workflow.Workflow.Status)
	require.Equal(t, 5, workflowAfterPayload.Workflow.Stats.Succeeded)

	opsResp, err := http.Get(ts.URL + "/api/v1/workflows/api-js-demo/ops")
	require.NoError(t, err)
	defer opsResp.Body.Close()
	require.Equal(t, http.StatusOK, opsResp.StatusCode)
	opsPayload := struct {
		WorkflowID string `json:"workflowID"`
		Ops        []struct {
			Status string `json:"status"`
			Op     struct {
				ID string `json:"id"`
			} `json:"op"`
		} `json:"ops"`
	}{}
	require.NoError(t, json.NewDecoder(opsResp.Body).Decode(&opsPayload))
	require.Equal(t, "api-js-demo", opsPayload.WorkflowID)
	require.Len(t, opsPayload.Ops, 5)
	require.Equal(t, "succeeded", opsPayload.Ops[0].Status)
}

func TestServerArtifactAndOpResultEndpoints(t *testing.T) {
	registry := sitesRegistry(t)

	ctx := context.Background()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	store, err := sqlitestore.Open(ctx, engineDB)
	require.NoError(t, err)

	now := time.Date(2026, 4, 7, 20, 0, 0, 0, time.UTC)
	require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:        "api-artifacts",
			Site:      "js-demo",
			Name:      "api artifacts",
			Status:    model.WorkflowStatusRunning,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Initial: []model.OpSpec{
			{ID: "api-artifacts:fetch", WorkflowID: "api-artifacts", Site: "js-demo", Kind: "http/fetch", Queue: "site:js-demo:http", DedupKey: "fetch"},
			{ID: "api-artifacts:extract", WorkflowID: "api-artifacts", Site: "js-demo", Kind: "js", Queue: "site:js-demo:js", DedupKey: "extract"},
		},
	}))
	require.NoError(t, completeServerTestOp(ctx, store, "js-demo", "site:js-demo:http", now, "frontpage.html", "http-response-body", "text/html", "<html>frontpage</html>", []byte(`{"url":"https://example.com"}`)))
	require.NoError(t, completeServerTestOp(ctx, store, "js-demo", "site:js-demo:js", now.Add(time.Second), "summary.json", "json-output", "application/json", `{"stories":30}`, []byte(`{"stories":30}`)))
	require.NoError(t, store.Close())

	server, err := apiserver.New(apiserver.Config{
		Address:      "127.0.0.1:0",
		EngineDB:     engineDB,
		SitesDir:     t.TempDir(),
		ReadTimeout:  5,
		WriteTimeout: 5,
		Version:      "test-version",
	}, registry)
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler)
	defer ts.Close()

	artifactsResp, err := http.Get(ts.URL + "/api/v1/workflows/api-artifacts/artifacts")
	require.NoError(t, err)
	defer artifactsResp.Body.Close()
	require.Equal(t, http.StatusOK, artifactsResp.StatusCode)

	artifactPayload := struct {
		WorkflowID string `json:"workflowID"`
		Total      int    `json:"total"`
		Artifacts  []struct {
			ID          string `json:"id"`
			OpID        string `json:"opID"`
			Name        string `json:"name"`
			Previewable bool   `json:"previewable"`
			PreviewKind string `json:"previewKind"`
		} `json:"artifacts"`
	}{}
	require.NoError(t, json.NewDecoder(artifactsResp.Body).Decode(&artifactPayload))
	require.Equal(t, "api-artifacts", artifactPayload.WorkflowID)
	require.Equal(t, 2, artifactPayload.Total)
	require.Len(t, artifactPayload.Artifacts, 2)
	require.Equal(t, "api-artifacts:fetch", artifactPayload.Artifacts[0].OpID)
	require.Equal(t, "frontpage.html", artifactPayload.Artifacts[0].Name)
	require.True(t, artifactPayload.Artifacts[0].Previewable)
	require.Equal(t, "html", artifactPayload.Artifacts[0].PreviewKind)

	filteredResp, err := http.Get(ts.URL + "/api/v1/workflows/api-artifacts/artifacts?opId=api-artifacts:extract&search=summary")
	require.NoError(t, err)
	defer filteredResp.Body.Close()
	require.Equal(t, http.StatusOK, filteredResp.StatusCode)

	filteredPayload := struct {
		Total     int `json:"total"`
		Artifacts []struct {
			Name string `json:"name"`
		} `json:"artifacts"`
	}{}
	require.NoError(t, json.NewDecoder(filteredResp.Body).Decode(&filteredPayload))
	require.Equal(t, 1, filteredPayload.Total)
	require.Len(t, filteredPayload.Artifacts, 1)
	require.Equal(t, "summary.json", filteredPayload.Artifacts[0].Name)

	resultResp, err := http.Get(ts.URL + "/api/v1/workflows/api-artifacts/ops/api-artifacts:extract/result")
	require.NoError(t, err)
	defer resultResp.Body.Close()
	require.Equal(t, http.StatusOK, resultResp.StatusCode)

	resultPayload := struct {
		Result *struct {
			OpID string          `json:"OpID"`
			Data json.RawMessage `json:"Data"`
		} `json:"result"`
	}{}
	require.NoError(t, json.NewDecoder(resultResp.Body).Decode(&resultPayload))
	require.NotNil(t, resultPayload.Result)
	require.Equal(t, "api-artifacts:extract", resultPayload.Result.OpID)
	require.JSONEq(t, `{"stories":30}`, string(resultPayload.Result.Data))

	notFoundResp, err := http.Get(ts.URL + "/api/v1/workflows/api-artifacts/ops/api-artifacts:missing/result")
	require.NoError(t, err)
	defer notFoundResp.Body.Close()
	require.Equal(t, http.StatusNotFound, notFoundResp.StatusCode)
}

func TestServerRuntimeEventsHistoryAndStream(t *testing.T) {
	registry := sitesRegistry(t)

	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	pubSub := runtimeevents.NewGoChannelPubSub()

	server, err := apiserver.New(apiserver.Config{
		Address:          "127.0.0.1:0",
		EngineDB:         engineDB,
		SitesDir:         sitesDir,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
		Version:          "test-version",
		RuntimeEvents:    runtimeevents.Config{Backend: runtimeevents.BackendGoChannel, GoChannel: pubSub},
		RecentEventLimit: 128,
	}, registry)
	require.NoError(t, err)

	ts := httptest.NewServer(server.Handler)
	defer ts.Close()

	waitForRuntimeEventRouter(t, ts.URL)

	body := bytes.NewBufferString(`{
		"workflowID": "event-js-demo",
		"values": {
			"count": 2,
			"multiplier": 3,
			"prefix": "events"
		}
	}`)
	resp, err := http.Post(ts.URL+"/api/v1/sites/js-demo/verbs/seed:submit", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	submitPayload := struct {
		Workflow struct {
			ID string `json:"id"`
		} `json:"workflow"`
	}{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&submitPayload))
	require.Equal(t, "event-js-demo", submitPayload.Workflow.ID)

	streamCtx, cancelStream := context.WithCancel(context.Background())
	defer cancelStream()
	streamURL := ts.URL + "/api/v1/runtime-events/stream?workflowId=" + url.QueryEscape(submitPayload.Workflow.ID)
	streamReq, err := http.NewRequestWithContext(streamCtx, http.MethodGet, streamURL, nil)
	require.NoError(t, err)
	streamResp, err := http.DefaultClient.Do(streamReq)
	require.NoError(t, err)
	defer streamResp.Body.Close()
	require.Equal(t, http.StatusOK, streamResp.StatusCode)

	streamEvents := make(chan *runtimeeventsEventEnvelope, 32)
	go collectRuntimeEvents(streamResp.Body, streamEvents)

	require.NoError(t, runWorkerWithRuntimeEvents(context.Background(), registry, engineDB, sitesDir, pubSub))

	var succeededSeen bool
	var runnerLogSeen bool
	timeout := time.After(10 * time.Second)
	for !(succeededSeen && runnerLogSeen) {
		select {
		case <-timeout:
			t.Fatalf("timed out waiting for streamed runtime events")
		case event := <-streamEvents:
			if event == nil {
				continue
			}
			if event.Kind == "RUNTIME_EVENT_KIND_OP_SUCCEEDED" {
				succeededSeen = true
			}
			if event.Kind == "RUNTIME_EVENT_KIND_LOG_LINE" && event.Source == "RUNTIME_EVENT_SOURCE_RUNNER" {
				runnerLogSeen = true
			}
		}
	}

	historyResp, err := http.Get(ts.URL + "/api/v1/runtime-events?workflowId=" + url.QueryEscape(submitPayload.Workflow.ID) + "&limit=64")
	require.NoError(t, err)
	defer historyResp.Body.Close()
	require.Equal(t, http.StatusOK, historyResp.StatusCode)

	historyPayload := struct {
		Events []json.RawMessage `json:"events"`
	}{}
	require.NoError(t, json.NewDecoder(historyResp.Body).Decode(&historyPayload))
	require.NotEmpty(t, historyPayload.Events)

	kinds := map[string]bool{}
	for _, raw := range historyPayload.Events {
		event, err := runtimeevents.UnmarshalJSON(raw)
		require.NoError(t, err)
		kinds[event.Kind.String()] = true
	}

	require.True(t, kinds["RUNTIME_EVENT_KIND_SUBMISSION_ACCEPTED"])
	require.True(t, kinds["RUNTIME_EVENT_KIND_WORKFLOW_CREATED"])
	require.True(t, kinds["RUNTIME_EVENT_KIND_OP_SUCCEEDED"])
	require.True(t, kinds["RUNTIME_EVENT_KIND_LOG_LINE"])
}

func completeServerTestOp(
	ctx context.Context,
	store *sqlitestore.Store,
	site model.SiteName,
	queue model.QueueKey,
	now time.Time,
	artifactName string,
	artifactKind string,
	artifactContentType string,
	artifactBody string,
	resultData []byte,
) error {
	op, lease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-test",
		Queue:         queue,
		Site:          site,
		LeaseDuration: 30 * time.Second,
		Now:           now,
	})
	if err != nil {
		return err
	}
	return store.CompleteOp(ctx, op.ID, storecontract.Completion{
		Lease: *lease,
		Result: model.OpResult{
			OpID: op.ID,
			Data: resultData,
			Artifacts: []model.ArtifactWrite{
				{
					ID:          model.ArtifactID(string(op.ID) + ":artifact"),
					Name:        artifactName,
					Kind:        artifactKind,
					ContentType: artifactContentType,
					Metadata:    map[string]string{"source": "test"},
					Body:        []byte(artifactBody),
				},
			},
			CompletedAt: now,
		},
	})
}

type runtimeeventsEventEnvelope struct {
	Kind   string `json:"kind"`
	Source string `json:"source"`
}

func collectRuntimeEvents(body io.Reader, events chan<- *runtimeeventsEventEnvelope) {
	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			close(events)
			return
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		event := &runtimeeventsEventEnvelope{}
		if err := json.Unmarshal([]byte(payload), event); err != nil {
			continue
		}
		events <- event
	}
}

func waitForRuntimeEventRouter(t *testing.T, baseURL string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/v1/runtime-events?limit=1")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("runtime event router did not become ready")
}

func runWorkerWithRuntimeEvents(
	ctx context.Context,
	registry *siteregistry.Registry,
	engineDB string,
	sitesDir string,
	pubSub *gochannel.GoChannel,
) error {
	if err := os.MkdirAll(filepath.Dir(engineDB), 0o755); err != nil {
		return err
	}

	store, err := sqlitestore.Open(ctx, engineDB)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	scraperDB, err := sql.Open("sqlite3", engineDB)
	if err != nil {
		return err
	}
	defer func() { _ = scraperDB.Close() }()

	eventPublisher := runtimeevents.NewPublisher(pubSub, "")
	runners := runner.NewRegistry()

	httpRunner, err := runner.NewHTTPRunner(config.HTTP{Timeout: 5 * time.Second}, nil)
	if err != nil {
		return err
	}
	if err := runners.Register(runtimeevents.WrapRunner(httpRunner, eventPublisher, "worker-runner", "test-worker")); err != nil {
		return err
	}
	if err := runners.Register(runtimeevents.WrapRunner(runner.NewJSRunner(registry), eventPublisher, "worker-runner", "test-worker")); err != nil {
		return err
	}

	s, err := scheduler.New(store, runners, scheduler.Config{
		MaxWorkers:           4,
		PollInterval:         5 * time.Millisecond,
		DefaultLeaseDuration: 30 * time.Second,
	}, "test-worker", runtimeevents.NewSchedulerObserver(eventPublisher, "worker-scheduler", "test-worker"))
	if err != nil {
		return err
	}
	s.SetScraperDB(scraperDB)
	s.SetQueuePolicyProvider(registry.QueuePolicyProvider())

	manager := sitemigrate.NewManager(registry)
	siteDBs := map[model.SiteName]*sql.DB{}
	defer func() {
		for _, db := range siteDBs {
			_ = db.Close()
		}
	}()
	s.SetSiteDBProvider(func(ctx context.Context, site model.SiteName) (databasemod.QueryExecer, error) {
		if db, ok := siteDBs[site]; ok {
			return db, nil
		}
		report, err := manager.Migrate(ctx, site, sitesDir)
		if err != nil {
			return nil, err
		}
		db, err := sql.Open("sqlite3", report.DatabasePath)
		if err != nil {
			return nil, err
		}
		siteDBs[site] = db
		return db, nil
	})

	for i := 0; i < 32; i++ {
		if _, err := s.RunOnce(ctx); err != nil {
			return err
		}
	}
	return nil
}

package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	apiserver "github.com/go-go-golems/scraper/pkg/api/server"
	scrapercmd "github.com/go-go-golems/scraper/pkg/cmd"
	"github.com/go-go-golems/scraper/pkg/sites/defaults"
	"github.com/stretchr/testify/require"
)

func TestServerHealthAndCatalogEndpoints(t *testing.T) {
	registry, err := defaults.NewRegistry()
	require.NoError(t, err)

	server := apiserver.New(apiserver.Config{
		Address:      "127.0.0.1:0",
		EngineDB:     t.TempDir() + "/engine.db",
		SitesDir:     t.TempDir(),
		ReadTimeout:  5,
		WriteTimeout: 5,
		Version:      "test-version",
	}, registry)

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
				Name string `json:"name"`
			} `json:"sites"`
		}{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		require.NotEmpty(t, payload.Sites)
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
	registry, err := defaults.NewRegistry()
	require.NoError(t, err)

	sitesDir := t.TempDir()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	server := apiserver.New(apiserver.Config{
		Address:      "127.0.0.1:0",
		EngineDB:     engineDB,
		SitesDir:     sitesDir,
		ReadTimeout:  5,
		WriteTimeout: 5,
		Version:      "test-version",
	}, registry)

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

	rootCmd, err := scrapercmd.NewRootCommand("test-version")
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

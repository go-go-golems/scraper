package runner

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/config"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/stretchr/testify/require"
)

func TestHTTPRunnerFetchSuccessWithTemplatesAndArtifact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "scraper/test", r.Header.Get("User-Agent"))
		require.Equal(t, "test-token", r.Header.Get("X-Test-Token"))
		require.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, "page=2&town=Milford", string(body))

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Result", "ok")
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte("<html>fixture page</html>"))
		require.NoError(t, err)
	}))
	defer server.Close()

	httpRunner := NewHTTPRunner(config.HTTP{
		UserAgent: "scraper/test",
		Timeout:   5 * time.Second,
	}, server.Client())

	result, err := httpRunner.Run(context.Background(), RunContext{
		Workflow: model.WorkflowRun{
			ID:    "wf-http",
			Site:  "nereval",
			Input: json.RawMessage(`{"defaultTown":"Milford"}`),
		},
		Op: model.OpSpec{
			ID:         "fetch-1",
			WorkflowID: "wf-http",
			Site:       "nereval",
			Kind:       "http/fetch",
			Queue:      "site:nereval:http",
			Input: json.RawMessage(`{
				"page": 2,
				"request": {
					"method": "POST",
					"url": "` + server.URL + `/list?page={{ .input.page }}",
					"headers": {
						"X-Test-Token": "test-token"
					},
					"form": {
						"town": "{{ .workflow.input.defaultTown }}",
						"page": "{{ .input.page }}"
					}
				},
				"persistBody": true,
				"artifactName": "fixture.html"
			}`),
		},
		Now: time.Date(2026, 3, 23, 18, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	require.Nil(t, result.Error)
	require.Len(t, result.Artifacts, 1)
	require.Equal(t, model.ArtifactID("fetch-1:response-body"), result.Artifacts[0].ID)
	require.Equal(t, "fixture.html", result.Artifacts[0].Name)
	require.Equal(t, "http-response-body", result.Artifacts[0].Kind)
	require.Equal(t, "text/html", result.Artifacts[0].ContentType)
	require.Equal(t, "<html>fixture page</html>", string(result.Artifacts[0].Body))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(result.Data, &envelope))

	requestEnvelope := envelope["request"].(map[string]any)
	require.Equal(t, "POST", requestEnvelope["method"])
	require.Equal(t, server.URL+"/list?page=2", requestEnvelope["url"])
	require.Equal(t, "page=2&town=Milford", requestEnvelope["bodyText"])
	require.Equal(t, "application/x-www-form-urlencoded", requestEnvelope["contentType"])

	requestHeaders := requestEnvelope["headers"].(map[string]any)
	require.Equal(t, []any{"test-token"}, requestHeaders["X-Test-Token"])

	responseEnvelope := envelope["response"].(map[string]any)
	require.Equal(t, float64(200), responseEnvelope["statusCode"])
	require.Equal(t, "200 OK", responseEnvelope["status"])
	require.Equal(t, server.URL+"/list?page=2", responseEnvelope["finalURL"])
	require.Equal(t, "text/html; charset=utf-8", responseEnvelope["contentType"])
	require.Equal(t, "fetch-1:response-body", responseEnvelope["bodyArtifactID"])

	responseHeaders := responseEnvelope["headers"].(map[string]any)
	require.Equal(t, []any{"text/html; charset=utf-8"}, responseHeaders["Content-Type"])
	require.Equal(t, []any{"ok"}, responseHeaders["X-Result"])
}

func TestHTTPRunnerMarksServerErrorsRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, err := w.Write([]byte(`{"error":"upstream failed"}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	httpRunner := NewHTTPRunner(config.HTTP{Timeout: 5 * time.Second}, server.Client())
	now := time.Date(2026, 3, 23, 18, 5, 0, 0, time.UTC)

	result, err := httpRunner.Run(context.Background(), RunContext{
		Workflow: model.WorkflowRun{ID: "wf-http", Site: "nereval"},
		Op: model.OpSpec{
			ID:         "fetch-502",
			WorkflowID: "wf-http",
			Site:       "nereval",
			Kind:       "http/fetch",
			Queue:      "site:nereval:http",
			Input:      json.RawMessage(`{"request":{"url":"` + server.URL + `"}}`),
		},
		Now: now,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Error)
	require.Equal(t, "http_status_502", result.Error.Code)
	require.True(t, result.Error.Retryable)
	require.Equal(t, now, result.Error.OccurredAt)
}

func TestHTTPRunnerMarksClientErrorsNonRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, err := w.Write([]byte("not found"))
		require.NoError(t, err)
	}))
	defer server.Close()

	httpRunner := NewHTTPRunner(config.HTTP{Timeout: 5 * time.Second}, server.Client())
	now := time.Date(2026, 3, 23, 18, 10, 0, 0, time.UTC)

	result, err := httpRunner.Run(context.Background(), RunContext{
		Workflow: model.WorkflowRun{ID: "wf-http", Site: "nereval"},
		Op: model.OpSpec{
			ID:         "fetch-404",
			WorkflowID: "wf-http",
			Site:       "nereval",
			Kind:       "http/fetch",
			Queue:      "site:nereval:http",
			Input:      json.RawMessage(`{"request":{"url":"` + server.URL + `"},"persistBody":true}`),
		},
		Now: now,
	})
	require.NoError(t, err)
	require.NotNil(t, result.Error)
	require.Equal(t, "http_status_404", result.Error.Code)
	require.False(t, result.Error.Retryable)
	require.Len(t, result.Artifacts, 1)
}

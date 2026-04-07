package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

type stubRunner struct {
	kind   string
	result *model.OpResult
	err    error
}

var _ runner.Runner = &stubRunner{}

func (r *stubRunner) Kind() string {
	return r.kind
}

func (r *stubRunner) Run(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
	return r.result, r.err
}

func TestObservedRunnerHTTPMetricsSuccess(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)

	base := &stubRunner{
		kind: "http/fetch",
		result: &model.OpResult{
			Data: mustMarshalJSON(t, map[string]any{
				"response": map[string]any{
					"statusCode": 204,
				},
			}),
		},
	}

	wrapped := WrapRunner(base, registry)
	runCtx := testRunContext()

	result, runErr := wrapped.Run(context.Background(), runCtx)
	require.NoError(t, runErr)
	require.NotNil(t, result)

	httpCount := testutil.ToFloat64(registry.HTTPRunnerRequestsTotal.WithLabelValues("js-demo", "default", "2xx"))
	require.Equal(t, 1.0, httpCount)

	completed := testutil.ToFloat64(registry.OpsCompletedTotal.WithLabelValues("js-demo", "default", "http/fetch", "succeeded"))
	require.Equal(t, 1.0, completed)
}

func TestObservedRunnerHTTPMetricsTransportError(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)

	base := &stubRunner{
		kind: "http/fetch",
		err:  errors.New("dial tcp: connection refused"),
	}

	wrapped := WrapRunner(base, registry)
	_, runErr := wrapped.Run(context.Background(), testRunContext())
	require.Error(t, runErr)

	httpCount := testutil.ToFloat64(registry.HTTPRunnerRequestsTotal.WithLabelValues("js-demo", "default", "transport_error"))
	require.Equal(t, 1.0, httpCount)

	completed := testutil.ToFloat64(registry.OpsCompletedTotal.WithLabelValues("js-demo", "default", "http/fetch", "error"))
	require.Equal(t, 1.0, completed)
}

func TestObservedRunnerHTTPMetricsRetryableFailure(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)

	base := &stubRunner{
		kind: "http/fetch",
		result: &model.OpResult{
			Error: &model.OpError{
				Code:      "http_5xx",
				Retryable: true,
			},
		},
	}

	wrapped := WrapRunner(base, registry)
	result, runErr := wrapped.Run(context.Background(), testRunContext())
	require.NoError(t, runErr)
	require.NotNil(t, result)

	httpCount := testutil.ToFloat64(registry.HTTPRunnerRequestsTotal.WithLabelValues("js-demo", "default", "5xx"))
	require.Equal(t, 1.0, httpCount)

	completed := testutil.ToFloat64(registry.OpsCompletedTotal.WithLabelValues("js-demo", "default", "http/fetch", "retried"))
	require.Equal(t, 1.0, completed)
}

func TestObservedRunnerHTTPMetricsNonRetryableFailure(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)

	base := &stubRunner{
		kind: "http/fetch",
		result: &model.OpResult{
			Error: &model.OpError{
				Code:      "http_4xx",
				Retryable: false,
			},
		},
	}

	wrapped := WrapRunner(base, registry)
	result, runErr := wrapped.Run(context.Background(), testRunContext())
	require.NoError(t, runErr)
	require.NotNil(t, result)

	httpCount := testutil.ToFloat64(registry.HTTPRunnerRequestsTotal.WithLabelValues("js-demo", "default", "4xx"))
	require.Equal(t, 1.0, httpCount)

	completed := testutil.ToFloat64(registry.OpsCompletedTotal.WithLabelValues("js-demo", "default", "http/fetch", "failed"))
	require.Equal(t, 1.0, completed)
}

func testRunContext() runner.RunContext {
	return runner.RunContext{
		Now: time.Unix(1, 0).UTC(),
		Op: model.OpSpec{
			ID:         "op-1",
			WorkflowID: "wf-1",
			Site:       "js-demo",
			Queue:      "default",
		},
	}
}

func mustMarshalJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()

	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return payload
}

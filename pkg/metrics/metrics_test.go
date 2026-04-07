package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryRegistersCollectors(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)
	require.NotNil(t, registry)
	require.NotNil(t, registry.PrometheusRegistry())
}

func TestObserveHTTPRequest(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)

	registry.ObserveHTTPRequest("GET", "/api/v1/queues", 200, 0)

	count := testutil.ToFloat64(registry.HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/queues", "2xx"))
	require.Equal(t, 1.0, count)
}

package submission

import (
	"context"
	"testing"

	"github.com/go-go-golems/scraper/pkg/metrics"
	"github.com/go-go-golems/scraper/pkg/sites/defaults"
	"github.com/go-go-golems/scraper/pkg/testfixtures"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestSubmitRecordsAcceptedMetrics(t *testing.T) {
	registry, err := defaults.NewRegistryFromDirs(testfixtures.SitesDir(t))
	require.NoError(t, err)

	metricsRegistry, err := metrics.NewRegistry()
	require.NoError(t, err)

	service := NewService(registry, nil, metricsRegistry)

	response, err := service.Submit(context.Background(), Request{
		Site:       "js-demo",
		Verb:       "seed",
		WorkflowID: "submission-metrics-accepted",
		EngineDB:   t.TempDir() + "/engine.db",
		SitesDir:   t.TempDir(),
		Values: map[string]interface{}{
			"count":      2,
			"multiplier": 3,
			"prefix":     "metrics",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, response)

	count := testutil.ToFloat64(metricsRegistry.WorkflowsSubmittedTotal.WithLabelValues("js-demo", "seed"))
	require.Equal(t, 1.0, count)
}

func TestSubmitRecordsNotFoundFailures(t *testing.T) {
	registry, err := defaults.NewRegistryFromDirs(testfixtures.SitesDir(t))
	require.NoError(t, err)

	metricsRegistry, err := metrics.NewRegistry()
	require.NoError(t, err)

	service := NewService(registry, nil, metricsRegistry)

	response, err := service.Submit(context.Background(), Request{
		Site: "missing-site",
		Verb: "seed",
	})
	require.Error(t, err)
	require.Nil(t, response)

	count := testutil.ToFloat64(metricsRegistry.SubmissionFailuresTotal.WithLabelValues("missing-site", "seed", "not_found"))
	require.Equal(t, 1.0, count)
}

func TestSubmitRecordsValidationFailures(t *testing.T) {
	registry, err := defaults.NewRegistryFromDirs(testfixtures.SitesDir(t))
	require.NoError(t, err)

	metricsRegistry, err := metrics.NewRegistry()
	require.NoError(t, err)

	service := NewService(registry, nil, metricsRegistry)

	response, err := service.Submit(context.Background(), Request{
		Site: "js-demo",
		Verb: "seed",
		Values: map[string]interface{}{
			"noSuchField": true,
		},
	})
	require.Error(t, err)
	require.Nil(t, response)

	count := testutil.ToFloat64(metricsRegistry.SubmissionFailuresTotal.WithLabelValues("js-demo", "seed", "validation_error"))
	require.Equal(t, 1.0, count)
}

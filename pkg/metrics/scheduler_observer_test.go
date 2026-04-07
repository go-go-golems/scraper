package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestSchedulerObserverTracksFailureCounters(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)

	observer := NewSchedulerObserver(registry)
	require.NotNil(t, observer)

	observer.OnSchedulerEvent(context.Background(), scheduler.Event{
		Kind:       scheduler.EventOpFailed,
		OccurredAt: time.Unix(1, 0).UTC(),
		Site:       "js-demo",
		Queue:      "default",
		RunnerKind: "http/fetch",
		Error: &model.OpError{
			Code:      "http_4xx",
			Retryable: false,
		},
	})

	count := testutil.ToFloat64(registry.OpFailuresTotal.WithLabelValues("js-demo", "default", "http/fetch", "http_4xx"))
	require.Equal(t, 1.0, count)
}

func TestSchedulerObserverTracksQueueWaitHistogram(t *testing.T) {
	registry, err := NewRegistry()
	require.NoError(t, err)

	observer := NewSchedulerObserver(registry)
	require.NotNil(t, observer)

	observer.OnSchedulerEvent(context.Background(), scheduler.Event{
		Kind:              scheduler.EventOpLeased,
		OccurredAt:        time.Unix(1, 0).UTC(),
		Site:              "js-demo",
		Queue:             "default",
		RunnerKind:        "http/fetch",
		QueueWaitDuration: 5 * time.Second,
	})

	histogramMetric, err := registry.QueueWaitDuration.GetMetricWithLabelValues("js-demo", "default", "http/fetch")
	require.NoError(t, err)

	metric := &dto.Metric{}
	require.NoError(t, histogramMetric.(interface{ Write(*dto.Metric) error }).Write(metric))
	require.Equal(t, uint64(1), metric.GetHistogram().GetSampleCount())
	require.InDelta(t, 5.0, metric.GetHistogram().GetSampleSum(), 0.001)
}

package metrics

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/go-go-golems/scraper/pkg/services/engineview"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestSnapshotCollectorExportsWorkflowStatusAndQueueMetrics(t *testing.T) {
	ctx := context.Background()
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	store, err := sqlitestore.Open(ctx, engineDB)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()

	now := time.Now().UTC()
	workflows := []model.WorkflowRun{
		{ID: "wf-pending", Site: "js-demo", Name: "Pending", Status: model.WorkflowStatusPending, CreatedAt: now, UpdatedAt: now},
		{ID: "wf-running", Site: "js-demo", Name: "Running", Status: model.WorkflowStatusRunning, CreatedAt: now, UpdatedAt: now},
		{ID: "wf-succeeded", Site: "js-demo", Name: "Succeeded", Status: model.WorkflowStatusSucceeded, CreatedAt: now, UpdatedAt: now},
	}

	for _, workflow := range workflows {
		initial := []model.OpSpec{}
		if workflow.ID == "wf-running" {
			initial = []model.OpSpec{
				{
					ID:         "op-running",
					WorkflowID: workflow.ID,
					Site:       workflow.Site,
					Kind:       "http/fetch",
					Queue:      "default",
					Retry: model.RetryPolicy{
						MaxAttempts:    1,
						BackoffKind:    model.BackoffKindFixed,
						InitialBackoff: time.Second,
						MaxBackoff:     time.Second,
						Multiplier:     1,
					},
				},
			}
		}
		require.NoError(t, store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
			Workflow: workflow,
			Initial:  initial,
		}))
	}

	_, _, err = store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID:      "worker-1",
		Queue:         "default",
		Site:          "js-demo",
		LeaseDuration: 30 * time.Second,
		Now:           now,
	})
	require.NoError(t, err)

	service := engineview.NewService(engineDB)
	collector := NewSnapshotCollector(service, time.Second)

	registry := prometheus.NewRegistry()
	require.NoError(t, registry.Register(collector))

	families, err := registry.Gather()
	require.NoError(t, err)

	requireMetricValue(t, families, "scraper_engine_workflows_total", nil, 3)
	requireMetricValue(t, families, "scraper_engine_workflow_status_total", map[string]string{"status": "pending"}, 1)
	requireMetricValue(t, families, "scraper_engine_workflow_status_total", map[string]string{"status": "running"}, 1)
	requireMetricValue(t, families, "scraper_engine_workflow_status_total", map[string]string{"status": "succeeded"}, 1)
	requireMetricValue(t, families, "scraper_queue_state_total", map[string]string{"site": "js-demo", "queue": "default", "state": "running"}, 1)
	requireMetricValue(t, families, "scraper_queue_state_total", map[string]string{"site": "js-demo", "queue": "default", "state": "in_flight"}, 1)
}

func requireMetricValue(t *testing.T, families []*dto.MetricFamily, familyName string, labels map[string]string, expected float64) {
	t.Helper()

	for _, family := range families {
		if family.GetName() != familyName {
			continue
		}
		for _, metric := range family.GetMetric() {
			if metricLabelsMatch(metric, labels) {
				require.Equal(t, expected, metric.GetGauge().GetValue())
				return
			}
		}
	}

	require.Failf(t, "metric not found", "family=%s labels=%v", familyName, labels)
}

func metricLabelsMatch(metric *dto.Metric, labels map[string]string) bool {
	if len(labels) == 0 {
		return len(metric.GetLabel()) == 0
	}
	if len(metric.GetLabel()) != len(labels) {
		return false
	}
	for _, label := range metric.GetLabel() {
		if labels[label.GetName()] != label.GetValue() {
			return false
		}
	}
	return true
}

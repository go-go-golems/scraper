package metrics

import (
	"context"
	"time"

	"github.com/go-go-golems/scraper/pkg/services/engineview"
	"github.com/prometheus/client_golang/prometheus"
)

type SnapshotCollector struct {
	service *engineview.Service
	timeout time.Duration

	workflowsTotal *prometheus.Desc
	workflowStatus *prometheus.Desc
	opStatusTotal  *prometheus.Desc
	leasesTotal    *prometheus.Desc
	resultsTotal   *prometheus.Desc
	artifactsTotal *prometheus.Desc
	queueState     *prometheus.Desc
	queueTokens    *prometheus.Desc
}

func NewSnapshotCollector(service *engineview.Service, timeout time.Duration) *SnapshotCollector {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &SnapshotCollector{
		service: service,
		timeout: timeout,
		workflowsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "engine_workflows_total"),
			"Current total workflows known to the engine.",
			nil,
			nil,
		),
		workflowStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "engine_workflow_status_total"),
			"Current total workflows in the engine grouped by workflow status.",
			[]string{"status"},
			nil,
		),
		opStatusTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "engine_ops_total"),
			"Current total ops in the engine grouped by status.",
			[]string{"status"},
			nil,
		),
		leasesTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "engine_leases_total"),
			"Current total leases in the engine grouped by lease state.",
			[]string{"state"},
			nil,
		),
		resultsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "engine_results_total"),
			"Current total stored op results in the engine.",
			nil,
			nil,
		),
		artifactsTotal: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "engine_artifacts_total"),
			"Current total stored artifacts in the engine.",
			nil,
			nil,
		),
		queueState: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "queue_state_total"),
			"Current queue state counts by site, queue, and state.",
			[]string{"site", "queue", "state"},
			nil,
		),
		queueTokens: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "queue_tokens"),
			"Current queue token-bucket fill by site and queue.",
			[]string{"site", "queue"},
			nil,
		),
	}
}

func (c *SnapshotCollector) Describe(ch chan<- *prometheus.Desc) {
	if c == nil {
		return
	}
	ch <- c.workflowsTotal
	ch <- c.workflowStatus
	ch <- c.opStatusTotal
	ch <- c.leasesTotal
	ch <- c.resultsTotal
	ch <- c.artifactsTotal
	ch <- c.queueState
	ch <- c.queueTokens
}

func (c *SnapshotCollector) Collect(ch chan<- prometheus.Metric) {
	if c == nil || c.service == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	status, err := c.service.EngineStatus(ctx)
	if err == nil && status != nil {
		ch <- prometheus.MustNewConstMetric(c.workflowsTotal, prometheus.GaugeValue, float64(status.WorkflowCount))
		for workflowStatus, count := range status.WorkflowCounts {
			ch <- prometheus.MustNewConstMetric(c.workflowStatus, prometheus.GaugeValue, float64(count), string(workflowStatus))
		}
		ch <- prometheus.MustNewConstMetric(c.resultsTotal, prometheus.GaugeValue, float64(status.ResultCount))
		ch <- prometheus.MustNewConstMetric(c.artifactsTotal, prometheus.GaugeValue, float64(status.ArtifactCount))
		ch <- prometheus.MustNewConstMetric(c.leasesTotal, prometheus.GaugeValue, float64(status.ActiveLeases), "active")
		ch <- prometheus.MustNewConstMetric(c.leasesTotal, prometheus.GaugeValue, float64(status.ExpiredLeases), "expired")
		for statusName, count := range status.OpCounts {
			ch <- prometheus.MustNewConstMetric(c.opStatusTotal, prometheus.GaugeValue, float64(count), string(statusName))
		}
	}

	queues, err := c.service.ListQueues(ctx)
	if err != nil {
		return
	}
	for _, queue := range queues {
		site := string(queue.Site)
		queueKey := string(queue.Queue)
		ch <- prometheus.MustNewConstMetric(c.queueState, prometheus.GaugeValue, float64(queue.Pending), site, queueKey, "pending")
		ch <- prometheus.MustNewConstMetric(c.queueState, prometheus.GaugeValue, float64(queue.Ready), site, queueKey, "ready")
		ch <- prometheus.MustNewConstMetric(c.queueState, prometheus.GaugeValue, float64(queue.Running), site, queueKey, "running")
		ch <- prometheus.MustNewConstMetric(c.queueState, prometheus.GaugeValue, float64(queue.InFlight), site, queueKey, "in_flight")
		ch <- prometheus.MustNewConstMetric(c.queueState, prometheus.GaugeValue, float64(queue.Succeeded), site, queueKey, "succeeded")
		ch <- prometheus.MustNewConstMetric(c.queueState, prometheus.GaugeValue, float64(queue.Failed), site, queueKey, "failed")
		if queue.Tokens != nil {
			ch <- prometheus.MustNewConstMetric(c.queueTokens, prometheus.GaugeValue, *queue.Tokens, site, queueKey)
		}
	}
}

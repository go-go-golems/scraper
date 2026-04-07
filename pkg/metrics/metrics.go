package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	namespace = "scraper"
)

var durationBuckets = []float64{
	0.005,
	0.01,
	0.025,
	0.05,
	0.1,
	0.25,
	0.5,
	1,
	2.5,
	5,
	10,
	30,
	60,
}

type Registry struct {
	registry *prometheus.Registry

	HTTPRequestsTotal       *prometheus.CounterVec
	HTTPRequestDuration     *prometheus.HistogramVec
	WorkflowsSubmittedTotal *prometheus.CounterVec
	SubmissionFailuresTotal *prometheus.CounterVec
	SchedulerCyclesTotal    *prometheus.CounterVec
	SchedulerCycleDuration  *prometheus.HistogramVec
	OpsLeasedTotal          *prometheus.CounterVec
	OpsCompletedTotal       *prometheus.CounterVec
	OpRetriesTotal          *prometheus.CounterVec
	QueueRateLimitedTotal   *prometheus.CounterVec
	OpDuration              *prometheus.HistogramVec
	HTTPRunnerRequestsTotal *prometheus.CounterVec
	HTTPRunnerDuration      *prometheus.HistogramVec
	WorkersUp               *prometheus.GaugeVec
}

func NewRegistry() (*Registry, error) {
	registry := prometheus.NewRegistry()
	if err := registry.Register(collectors.NewGoCollector()); err != nil {
		return nil, err
	}
	if err := registry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{})); err != nil {
		return nil, err
	}

	ret := &Registry{
		registry: registry,
		HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total HTTP requests served by the scraper API server.",
		}, []string{"method", "route", "status_class"}),
		HTTPRequestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration for the scraper API server.",
			Buckets:   durationBuckets,
		}, []string{"method", "route", "status_class"}),
		WorkflowsSubmittedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "workflows_submitted_total",
			Help:      "Total workflow submissions accepted by the submission service.",
		}, []string{"site", "verb"}),
		SubmissionFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "submission_failures_total",
			Help:      "Total workflow submission failures by site, verb, and error code.",
		}, []string{"site", "verb", "error_code"}),
		SchedulerCyclesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "scheduler_cycles_total",
			Help:      "Total scheduler cycles executed by a worker.",
		}, []string{"worker_id"}),
		SchedulerCycleDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "scheduler_cycle_duration_seconds",
			Help:      "Duration of scheduler cycles executed by a worker.",
			Buckets:   durationBuckets,
		}, []string{"worker_id"}),
		OpsLeasedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "ops_leased_total",
			Help:      "Total ops leased by site, queue, and runner kind.",
		}, []string{"site", "queue", "runner"}),
		OpsCompletedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "ops_completed_total",
			Help:      "Total ops completed by site, queue, runner kind, and terminal status.",
		}, []string{"site", "queue", "runner", "status"}),
		OpRetriesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "op_retries_total",
			Help:      "Total retried ops by site, queue, and runner kind.",
		}, []string{"site", "queue", "runner"}),
		QueueRateLimitedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "queue_rate_limited_total",
			Help:      "Total queue rate-limit events by site and queue.",
		}, []string{"site", "queue"}),
		OpDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "op_duration_seconds",
			Help:      "Duration of op execution by site, queue, runner kind, and terminal status.",
			Buckets:   durationBuckets,
		}, []string{"site", "queue", "runner", "status"}),
		HTTPRunnerRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_runner_requests_total",
			Help:      "Total HTTP runner requests by site, queue, and response class.",
		}, []string{"site", "queue", "status_class"}),
		HTTPRunnerDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_runner_duration_seconds",
			Help:      "Duration of HTTP runner requests by site, queue, and response class.",
			Buckets:   durationBuckets,
		}, []string{"site", "queue", "status_class"}),
		WorkersUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "workers_up",
			Help:      "Worker liveness gauge for workers exposing a metrics listener.",
		}, []string{"worker_id"}),
	}

	for _, collector := range []prometheus.Collector{
		ret.HTTPRequestsTotal,
		ret.HTTPRequestDuration,
		ret.WorkflowsSubmittedTotal,
		ret.SubmissionFailuresTotal,
		ret.SchedulerCyclesTotal,
		ret.SchedulerCycleDuration,
		ret.OpsLeasedTotal,
		ret.OpsCompletedTotal,
		ret.OpRetriesTotal,
		ret.QueueRateLimitedTotal,
		ret.OpDuration,
		ret.HTTPRunnerRequestsTotal,
		ret.HTTPRunnerDuration,
		ret.WorkersUp,
	} {
		if err := registry.Register(collector); err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (r *Registry) PrometheusRegistry() *prometheus.Registry {
	if r == nil {
		return nil
	}
	return r.registry
}

func (r *Registry) RegisterCollector(collector prometheus.Collector) error {
	if r == nil || r.registry == nil || collector == nil {
		return nil
	}
	return r.registry.Register(collector)
}

func (r *Registry) Handler() http.Handler {
	if r == nil || r.registry == nil {
		return promhttp.Handler()
	}
	return promhttp.HandlerFor(r.registry, promhttp.HandlerOpts{})
}

func StatusClass(code int) string {
	if code < 100 {
		return "unknown"
	}
	return strconv.Itoa(code/100) + "xx"
}

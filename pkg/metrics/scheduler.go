package metrics

import (
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

func (r *Registry) ObserveSchedulerCycle(workerID string, duration time.Duration) {
	if r == nil {
		return
	}
	r.SchedulerCyclesTotal.WithLabelValues(workerID).Inc()
	r.SchedulerCycleDuration.WithLabelValues(workerID).Observe(duration.Seconds())
}

func (r *Registry) ObserveOpLeased(site model.SiteName, queue model.QueueKey, runnerKind string) {
	if r == nil {
		return
	}
	r.OpsLeasedTotal.WithLabelValues(string(site), string(queue), runnerKind).Inc()
}

func (r *Registry) ObserveOpRetried(site model.SiteName, queue model.QueueKey, runnerKind string) {
	if r == nil {
		return
	}
	r.OpRetriesTotal.WithLabelValues(string(site), string(queue), runnerKind).Inc()
}

func (r *Registry) ObserveQueueRateLimited(site model.SiteName, queue model.QueueKey) {
	if r == nil {
		return
	}
	r.QueueRateLimitedTotal.WithLabelValues(string(site), string(queue)).Inc()
}

func (r *Registry) ObserveOpCompleted(site model.SiteName, queue model.QueueKey, runnerKind string, status string, duration time.Duration) {
	if r == nil {
		return
	}
	if status == "" {
		status = "unknown"
	}
	r.OpsCompletedTotal.WithLabelValues(string(site), string(queue), runnerKind, status).Inc()
	r.OpDuration.WithLabelValues(string(site), string(queue), runnerKind, status).Observe(duration.Seconds())
}

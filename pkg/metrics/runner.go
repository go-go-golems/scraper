package metrics

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
)

type ObservedRunner struct {
	base     runner.Runner
	registry *Registry
}

func WrapRunner(base runner.Runner, registry *Registry) runner.Runner {
	if base == nil || registry == nil {
		return base
	}
	return &ObservedRunner{
		base:     base,
		registry: registry,
	}
}

func (r *ObservedRunner) Kind() string {
	return r.base.Kind()
}

func (r *ObservedRunner) Run(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
	startedAt := time.Now().UTC()
	result, err := r.base.Run(ctx, runCtx)
	completedAt := time.Now().UTC()

	status := "error"
	if err == nil {
		status = "succeeded"
		if result != nil && result.Error != nil {
			if result.Error.Retryable {
				status = "retried"
			} else {
				status = "failed"
			}
		}
	}

	r.registry.ObserveOpCompleted(runCtx.Op.Site, runCtx.Op.Queue, r.base.Kind(), status, completedAt.Sub(startedAt))
	if r.base.Kind() == "http/fetch" {
		r.registry.ObserveHTTPRunner(runCtx.Op.Site, runCtx.Op.Queue, httpRunnerStatusClass(result, err), completedAt.Sub(startedAt))
	}
	return result, err
}

func (r *Registry) ObserveHTTPRunner(site model.SiteName, queue model.QueueKey, statusClass string, duration time.Duration) {
	if r == nil {
		return
	}
	if statusClass == "" {
		statusClass = "unknown"
	}
	r.HTTPRunnerRequestsTotal.WithLabelValues(string(site), string(queue), statusClass).Inc()
	r.HTTPRunnerDuration.WithLabelValues(string(site), string(queue), statusClass).Observe(duration.Seconds())
}

func httpRunnerStatusClass(result *model.OpResult, err error) string {
	if err != nil {
		return "transport_error"
	}
	if result == nil {
		return "unknown"
	}
	if result.Error != nil {
		switch result.Error.Code {
		case "transport_error", "read_response_error":
			return "transport_error"
		case "http_4xx":
			return "4xx"
		case "http_5xx":
			return "5xx"
		}
	}

	var envelope struct {
		Response struct {
			StatusCode int `json:"statusCode"`
		} `json:"response"`
	}
	if len(result.Data) == 0 {
		return "unknown"
	}
	if err := json.Unmarshal(result.Data, &envelope); err != nil {
		return "unknown"
	}
	if envelope.Response.StatusCode == 0 {
		return "unknown"
	}
	return StatusClass(envelope.Response.StatusCode)
}

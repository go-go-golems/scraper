package metrics

import "time"

func (r *Registry) ObserveHTTPRequest(method string, route string, statusCode int, duration time.Duration) {
	if r == nil {
		return
	}
	if route == "" {
		route = "unknown"
	}
	statusClass := StatusClass(statusCode)
	r.HTTPRequestsTotal.WithLabelValues(method, route, statusClass).Inc()
	r.HTTPRequestDuration.WithLabelValues(method, route, statusClass).Observe(duration.Seconds())
}

package server

import (
	"net/http"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/go-go-golems/scraper/pkg/metrics"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	"github.com/rs/zerolog/log"
)

func requestLogger(next http.Handler, eventPublisher *runtimeevents.Publisher, metricsRegistry *metrics.Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if requestID == "" {
			requestID = watermill.NewShortUUID()
		}
		r = r.WithContext(runtimeevents.ContextWithRequestID(r.Context(), requestID))

		_ = runtimeevents.EmitSimpleEvent(eventPublisher, &runtimev1.RuntimeEventV1{
			Source:    runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_REQUEST,
			Component: "http-api",
			Kind:      runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_REQUEST_RECEIVED,
			Severity:  runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_DEBUG,
			Message:   "request received",
			RequestId: requestID,
			Payload: runtimeevents.BuildPayload(map[string]any{
				"method": r.Method,
				"path":   r.URL.Path,
			}),
		})

		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		duration := time.Since(started)
		route := r.Pattern
		if route == "" {
			route = r.URL.Path
		}
		metricsRegistry.ObserveHTTPRequest(r.Method, route, recorder.status, duration)

		_ = runtimeevents.EmitSimpleEvent(eventPublisher, &runtimev1.RuntimeEventV1{
			Source:    runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_REQUEST,
			Component: "http-api",
			Kind:      runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_REQUEST_SERVED,
			Severity:  runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_DEBUG,
			Message:   "request served",
			RequestId: requestID,
			Payload: runtimeevents.BuildPayload(map[string]any{
				"method":         r.Method,
				"path":           r.URL.Path,
				"statusCode":     recorder.status,
				"durationMillis": duration.Milliseconds(),
			}),
		})

		log.Info().
			Str("component", "http-api").
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Dur("duration", duration).
			Msg("served request")
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

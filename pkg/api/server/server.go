package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/go-go-golems/scraper/pkg/api/handlers"
	"github.com/go-go-golems/scraper/pkg/metrics"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	"github.com/go-go-golems/scraper/pkg/services/catalog"
	"github.com/go-go-golems/scraper/pkg/services/engineview"
	"github.com/go-go-golems/scraper/pkg/services/submission"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Address          string
	EngineDB         string
	SitesDir         string
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	Version          string
	RuntimeEvents    runtimeevents.Config
	RecentEventLimit int
}

func New(cfg Config, siteRegistry *siteregistry.Registry) (*http.Server, error) {
	eventResources, err := runtimeevents.OpenPublisherSubscriber(cfg.RuntimeEvents)
	if err != nil {
		return nil, err
	}
	eventPublisher := eventResources.EventPublisher()
	eventHub := runtimeevents.NewHub(cfg.RecentEventLimit)

	router, err := startRuntimeEventRouter(eventResources, eventHub)
	if err != nil {
		_ = eventResources.Close()
		return nil, err
	}

	catalogService := catalog.NewService(siteRegistry)
	engineService := engineview.NewService(cfg.EngineDB)
	metricsRegistry, err := metrics.NewRegistry()
	if err != nil {
		_ = eventResources.Close()
		return nil, err
	}
	if err := metricsRegistry.RegisterCollector(metrics.NewSnapshotCollector(engineService, 2*time.Second)); err != nil {
		_ = eventResources.Close()
		return nil, err
	}
	submissionService := submission.NewService(siteRegistry, eventPublisher, metricsRegistry)

	catalogHandler := handlers.NewCatalogHandler(catalogService, cfg.Version, cfg.Address, cfg.EngineDB, cfg.SitesDir)
	submissionHandler := handlers.NewSubmissionHandler(submissionService, cfg.EngineDB, cfg.SitesDir, eventPublisher)
	engineHandler := handlers.NewEngineHandler(engineService, catalogService)
	runtimeEventsHandler := handlers.NewRuntimeEventsHandler(eventHub)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", catalogHandler.Healthz)
	mux.Handle("GET /metrics", metricsRegistry.Handler())
	mux.HandleFunc("GET /api/v1/info", catalogHandler.Info)
	mux.HandleFunc("GET /api/v1/sites", catalogHandler.Sites)
	mux.HandleFunc("GET /api/v1/sites/{site}", catalogHandler.Site)
	mux.HandleFunc("GET /api/v1/sites/{site}/detail", catalogHandler.SiteDetail)
	mux.HandleFunc("GET /api/v1/sites/{site}/verbs", catalogHandler.Verbs)
	mux.HandleFunc("GET /api/v1/sites/{site}/verbs/{verb}", catalogHandler.Verb)
	mux.HandleFunc("GET /api/v1/engine/status", engineHandler.Status)
	mux.HandleFunc("GET /api/v1/engine/migrations", engineHandler.Migrations)
	mux.HandleFunc("GET /api/v1/workflows", engineHandler.Workflows)
	mux.HandleFunc("GET /api/v1/workflows/{workflowID}", engineHandler.Workflow)
	mux.HandleFunc("GET /api/v1/workflows/{workflowID}/ops", engineHandler.WorkflowOps)
	mux.HandleFunc("GET /api/v1/workflows/{workflowID}/ops/{opID}/artifacts", engineHandler.OpArtifacts)
	mux.HandleFunc("GET /api/v1/artifacts/{artifactID}", engineHandler.ArtifactDownload)
	mux.HandleFunc("GET /api/v1/runtime-events", runtimeEventsHandler.List)
	mux.HandleFunc("GET /api/v1/runtime-events/stream", runtimeEventsHandler.Stream)
	mux.HandleFunc("POST /api/v1/workflows/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/workflows/")
		if strings.HasSuffix(path, ":cancel") {
			wfID := strings.TrimSuffix(path, ":cancel")
			r.SetPathValue("workflowID", wfID)
			engineHandler.CancelWorkflow(w, r)
			return
		}
		parts := strings.Split(path, "/")
		if len(parts) == 3 && parts[1] == "ops" && strings.HasSuffix(parts[2], ":retry") {
			r.SetPathValue("workflowID", parts[0])
			r.SetPathValue("opID", strings.TrimSuffix(parts[2], ":retry"))
			engineHandler.RetryOp(w, r)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("GET /api/v1/queues", engineHandler.Queues)
	mux.HandleFunc("GET /api/v1/sites/{site}/scripts", catalogHandler.Scripts)
	mux.HandleFunc("GET /api/v1/sites/{site}/scripts/{path...}", catalogHandler.Script)
	mux.HandleFunc("POST /api/v1/sites/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ":submit") {
			http.NotFound(w, r)
			return
		}
		trimmed := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/sites/"), ":submit")
		parts := strings.Split(trimmed, "/")
		if len(parts) != 3 || parts[1] != "verbs" {
			http.NotFound(w, r)
			return
		}
		r.SetPathValue("site", parts[0])
		r.SetPathValue("verb", parts[2])
		submissionHandler.Submit(w, r)
	})

	server := &http.Server{
		Addr:         cfg.Address,
		Handler:      requestLogger(mux, eventPublisher, metricsRegistry),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
	server.RegisterOnShutdown(func() {
		if router != nil {
			_ = router.Close()
		}
		_ = eventResources.Close()
	})
	return server, nil
}

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

func startRuntimeEventRouter(resources *runtimeevents.Resources, hub *runtimeevents.Hub) (*message.Router, error) {
	if resources == nil || resources.Subscriber == nil {
		return nil, nil
	}

	router, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, err
	}
	router.AddNoPublisherHandler("runtime-events-hub", resources.Topic, resources.Subscriber, func(msg *message.Message) error {
		event, err := runtimeevents.EventFromMessage(msg)
		if err != nil {
			return err
		}
		hub.Add(event)
		return nil
	})

	go func() {
		if err := router.Run(context.Background()); err != nil {
			log.Warn().Err(err).Msg("runtime event router stopped")
		}
	}()
	return router, nil
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

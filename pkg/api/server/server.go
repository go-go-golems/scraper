package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-go-golems/scraper/pkg/api/handlers"
	"github.com/go-go-golems/scraper/pkg/services/catalog"
	"github.com/go-go-golems/scraper/pkg/services/engineview"
	"github.com/go-go-golems/scraper/pkg/services/submission"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Address      string
	EngineDB     string
	SitesDir     string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	Version      string
}

func New(cfg Config, siteRegistry *siteregistry.Registry) *http.Server {
	catalogService := catalog.NewService(siteRegistry)
	submissionService := submission.NewService(siteRegistry)
	engineService := engineview.NewService(cfg.EngineDB)

	catalogHandler := handlers.NewCatalogHandler(catalogService, cfg.Version, cfg.Address, cfg.EngineDB, cfg.SitesDir)
	submissionHandler := handlers.NewSubmissionHandler(submissionService, cfg.EngineDB, cfg.SitesDir)
	engineHandler := handlers.NewEngineHandler(engineService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", catalogHandler.Healthz)
	mux.HandleFunc("GET /api/v1/info", catalogHandler.Info)
	mux.HandleFunc("GET /api/v1/sites", catalogHandler.Sites)
	mux.HandleFunc("GET /api/v1/sites/{site}", catalogHandler.Site)
	mux.HandleFunc("GET /api/v1/sites/{site}/verbs", catalogHandler.Verbs)
	mux.HandleFunc("GET /api/v1/sites/{site}/verbs/{verb}", catalogHandler.Verb)
	mux.HandleFunc("GET /api/v1/engine/status", engineHandler.Status)
	mux.HandleFunc("GET /api/v1/engine/migrations", engineHandler.Migrations)
	mux.HandleFunc("GET /api/v1/workflows/{workflowID}", engineHandler.Workflow)
	mux.HandleFunc("GET /api/v1/workflows/{workflowID}/ops", engineHandler.WorkflowOps)
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

	return &http.Server{
		Addr:         cfg.Address,
		Handler:      requestLogger(mux),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Info().
			Str("component", "http-api").
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Dur("duration", time.Since(started)).
			Msg("served request")
	})
}

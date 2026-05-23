package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-go-golems/scraper/pkg/api/handlers"
	"github.com/go-go-golems/scraper/pkg/metrics"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	runtimestream "github.com/go-go-golems/scraper/pkg/runtimeevents/sessionstream"
	"github.com/go-go-golems/scraper/pkg/services/catalog"
	"github.com/go-go-golems/scraper/pkg/services/engineview"
	"github.com/go-go-golems/scraper/pkg/services/submission"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
)

type Config struct {
	Address          string
	EngineDB         string
	SitesDir         string
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	Version          string
	RuntimeEvents    runtimeevents.Config
	RuntimeEventsDB  string
	RecentEventLimit int
}

func New(cfg Config, siteRegistry *siteregistry.Registry) (*http.Server, error) {
	runtimeStream, err := runtimestream.NewServerRuntime(context.Background(), runtimestream.Config{
		Events:      cfg.RuntimeEvents,
		TimelineDB:  cfg.RuntimeEventsDB,
		RecentLimit: cfg.RecentEventLimit,
	})
	if err != nil {
		return nil, err
	}
	eventPublisher := runtimeStream.Publisher

	catalogService := catalog.NewService(siteRegistry)
	engineService := engineview.NewService(cfg.EngineDB)
	metricsRegistry, err := metrics.NewRegistry()
	if err != nil {
		_ = runtimeStream.Close(context.Background())
		return nil, err
	}
	if err := metricsRegistry.RegisterCollector(metrics.NewSnapshotCollector(engineService, 2*time.Second)); err != nil {
		_ = runtimeStream.Close(context.Background())
		return nil, err
	}
	submissionService := submission.NewService(siteRegistry, eventPublisher, metricsRegistry)

	catalogHandler := handlers.NewCatalogHandler(catalogService, cfg.Version, cfg.Address, cfg.EngineDB, cfg.SitesDir)
	submissionHandler := handlers.NewSubmissionHandler(submissionService, cfg.EngineDB, cfg.SitesDir, eventPublisher)
	engineHandler := handlers.NewEngineHandler(engineService, catalogService)

	mux := http.NewServeMux()
	registerCatalogRoutes(mux, catalogHandler, submissionHandler)
	registerEngineRoutes(mux, engineHandler, metricsRegistry)
	registerRuntimeEventRoutes(mux, runtimeStream.WSServer)

	server := &http.Server{
		Addr:         cfg.Address,
		Handler:      requestLogger(mux, eventPublisher, metricsRegistry),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
	server.RegisterOnShutdown(func() {
		_ = runtimeStream.Close(context.Background())
	})
	return server, nil
}

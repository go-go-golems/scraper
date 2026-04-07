package server

import (
	"context"
	"net/http"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
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
	registerCatalogRoutes(mux, catalogHandler, submissionHandler)
	registerEngineRoutes(mux, engineHandler, metricsRegistry)
	registerRuntimeEventRoutes(mux, runtimeEventsHandler)

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

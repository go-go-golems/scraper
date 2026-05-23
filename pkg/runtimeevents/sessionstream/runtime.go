package runtimestream

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	storesqlite "github.com/go-go-golems/sessionstream/pkg/sessionstream/hydration/sqlite"
	ws "github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws"
)

type Runtime struct {
	Registry  *sessionstream.SchemaRegistry
	Store     sessionstream.HydrationStore
	Hub       *sessionstream.Hub
	WSServer  *ws.Server
	Publisher *Publisher

	resources  *runtimeevents.Resources
	closeStore func() error
}

type Config struct {
	Events      runtimeevents.Config
	TimelineDB  string
	RecentLimit int
}

func NewProducerRuntime(cfg Config) (*Runtime, error) {
	reg, err := newRegisteredSchemaRegistry()
	if err != nil {
		return nil, err
	}
	resources, err := runtimeevents.OpenPublisher(normalizeEventConfig(cfg.Events))
	if err != nil {
		return nil, err
	}
	hubOptions := []sessionstream.HubOption{sessionstream.WithSchemaRegistry(reg)}
	if resources.Publisher != nil {
		hubOptions = append(hubOptions, sessionstream.WithEventBus(resources.Publisher, noopSubscriber{}, sessionstream.WithBusTopic(resources.Topic)))
	}
	hub, err := sessionstream.NewHub(hubOptions...)
	if err != nil {
		_ = resources.Close()
		return nil, err
	}
	if err := RegisterCommands(hub); err != nil {
		_ = resources.Close()
		return nil, err
	}
	return &Runtime{Registry: reg, Hub: hub, Publisher: NewPublisher(hub), resources: resources}, nil
}

func NewServerRuntime(ctx context.Context, cfg Config) (*Runtime, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	reg, err := newRegisteredSchemaRegistry()
	if err != nil {
		return nil, err
	}
	store, closeStore, err := openHydrationStore(cfg.TimelineDB, reg)
	if err != nil {
		return nil, err
	}
	resources, err := runtimeevents.OpenPublisherSubscriber(normalizeEventConfig(cfg.Events))
	if err != nil {
		_ = closeStore()
		return nil, err
	}

	provider := &snapshotProvider{}
	wsServer, err := ws.NewServer(provider)
	if err != nil {
		_ = resources.Close()
		_ = closeStore()
		return nil, err
	}

	hubOptions := []sessionstream.HubOption{
		sessionstream.WithSchemaRegistry(reg),
		sessionstream.WithHydrationStore(store),
		sessionstream.WithUIFanout(wsServer),
		sessionstream.WithProjectionErrorPolicy(sessionstream.ProjectionErrorPolicyAdvance),
	}
	if resources.Publisher != nil && resources.Subscriber != nil {
		hubOptions = append(hubOptions, sessionstream.WithEventBus(resources.Publisher, resources.Subscriber, sessionstream.WithBusTopic(resources.Topic)))
	}
	hub, err := sessionstream.NewHub(hubOptions...)
	if err != nil {
		_ = resources.Close()
		_ = closeStore()
		return nil, err
	}
	provider.hub = hub
	if err := Install(hub, cfg.RecentLimit); err != nil {
		_ = resources.Close()
		_ = closeStore()
		return nil, err
	}
	if err := hub.Run(ctx); err != nil {
		_ = resources.Close()
		_ = closeStore()
		return nil, err
	}
	return &Runtime{Registry: reg, Store: store, Hub: hub, WSServer: wsServer, Publisher: NewPublisher(hub), resources: resources, closeStore: closeStore}, nil
}

func Install(hub *sessionstream.Hub, recentLimit int) error {
	if hub == nil {
		return fmt.Errorf("hub is nil")
	}
	if err := RegisterCommands(hub); err != nil {
		return err
	}
	if err := hub.RegisterUIProjection(UIProjection{}); err != nil {
		return err
	}
	projection := DefaultTimelineProjection()
	if recentLimit > 0 {
		projection.MaxEntitiesPerSession = recentLimit
	}
	return hub.RegisterTimelineProjection(projection)
}

func (r *Runtime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}
	var err error
	if r.Hub != nil {
		err = r.Hub.Shutdown(ctx)
	}
	if r.resources != nil {
		err = joinErrors(err, r.resources.Close())
	}
	if r.closeStore != nil {
		err = joinErrors(err, r.closeStore())
	}
	return err
}

func newRegisteredSchemaRegistry() (*sessionstream.SchemaRegistry, error) {
	reg := sessionstream.NewSchemaRegistry()
	if err := RegisterSchemas(reg); err != nil {
		return nil, err
	}
	return reg, nil
}

func normalizeEventConfig(cfg runtimeevents.Config) runtimeevents.Config {
	if cfg.Topic == "" {
		cfg.Topic = TopicRuntimeEventsSessionstreamV1
	}
	return cfg
}

func openHydrationStore(path string, reg *sessionstream.SchemaRegistry) (sessionstream.HydrationStore, func() error, error) {
	if path == "" {
		store, err := storesqlite.NewInMemory(reg)
		if err != nil {
			return nil, nil, err
		}
		return store, store.Close, nil
	}
	dsn, err := storesqlite.FileDSN(path)
	if err != nil {
		return nil, nil, err
	}
	store, err := storesqlite.New(dsn, reg)
	if err != nil {
		return nil, nil, err
	}
	return store, store.Close, nil
}

type snapshotProvider struct {
	hub *sessionstream.Hub
}

func (p *snapshotProvider) Snapshot(ctx context.Context, sid sessionstream.SessionId) (sessionstream.Snapshot, error) {
	if p == nil || p.hub == nil {
		return sessionstream.Snapshot{}, fmt.Errorf("runtime event hub is not initialized")
	}
	return p.hub.Snapshot(ctx, sid)
}

type noopSubscriber struct{}

func (noopSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	ch := make(chan *message.Message)
	close(ch)
	return ch, nil
}

func (noopSubscriber) Close() error { return nil }

func joinErrors(a, b error) error {
	if a != nil && b != nil {
		return fmt.Errorf("%v; %w", a, b)
	}
	if a != nil {
		return a
	}
	return b
}

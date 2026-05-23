package runtimestream

import (
	"context"
	"sync"
	"testing"
	"time"

	streamv1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/sessionstream/v1"
	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	storesqlite "github.com/go-go-golems/sessionstream/pkg/sessionstream/hydration/sqlite"
	"github.com/stretchr/testify/require"
)

func TestPublisherProjectsIntoLocalHubSnapshots(t *testing.T) {
	runtime := newTestRuntime(t, nil)
	defer func() { require.NoError(t, runtime.Close(context.Background())) }()

	require.NoError(t, runtime.Publisher.Publish(context.Background(), &runtimev1.RuntimeEventV1{WorkflowId: "wf-1", Message: "started"}))

	snap, err := runtime.Hub.Snapshot(context.Background(), SessionRuntimeGlobal)
	require.NoError(t, err)
	require.Len(t, snap.Entities, 1)
	entity, ok := snap.Entities[0].Payload.(*streamv1.RuntimeEventEntity)
	require.True(t, ok)
	require.NotEmpty(t, entity.GetEvent().GetId())
	require.Equal(t, uint32(runtimeevents.SchemaVersionV1), entity.GetEvent().GetSchemaVersion())
	require.NotNil(t, entity.GetEvent().GetOccurredAt())
	require.Equal(t, "started", entity.GetEvent().GetMessage())

	workflowSnap, err := runtime.Hub.Snapshot(context.Background(), WorkflowSessionID("wf-1"))
	require.NoError(t, err)
	require.Len(t, workflowSnap.Entities, 1)
}

func TestProducerAndServerRuntimeFanoutOverGoChannel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pubSub := runtimeevents.NewGoChannelPubSub()
	server, err := NewServerRuntime(ctx, Config{Events: runtimeevents.Config{Backend: runtimeevents.BackendGoChannel, GoChannel: pubSub}, RecentLimit: 16})
	require.NoError(t, err)
	defer func() { require.NoError(t, server.Close(context.Background())) }()

	producer, err := NewProducerRuntime(Config{Events: runtimeevents.Config{Backend: runtimeevents.BackendGoChannel, GoChannel: pubSub}})
	require.NoError(t, err)
	defer func() { require.NoError(t, producer.Close(context.Background())) }()

	require.NoError(t, producer.Publisher.Publish(context.Background(), &runtimev1.RuntimeEventV1{WorkflowId: "wf-2", Message: "from worker"}))

	require.Eventually(t, func() bool {
		snap, err := server.Hub.Snapshot(context.Background(), WorkflowSessionID("wf-2"))
		return err == nil && len(snap.Entities) == 1
	}, 2*time.Second, 10*time.Millisecond)
}

func newTestRuntime(t *testing.T, fanout sessionstream.UIFanout) *Runtime {
	t.Helper()
	reg, err := newRegisteredSchemaRegistry()
	require.NoError(t, err)
	store, err := storesqlite.NewInMemory(reg)
	require.NoError(t, err)
	hubOptions := []sessionstream.HubOption{
		sessionstream.WithSchemaRegistry(reg),
		sessionstream.WithHydrationStore(store),
	}
	if fanout != nil {
		hubOptions = append(hubOptions, sessionstream.WithUIFanout(fanout))
	}
	hub, err := sessionstream.NewHub(hubOptions...)
	require.NoError(t, err)
	require.NoError(t, Install(hub, 16))
	return &Runtime{Registry: reg, Store: store, Hub: hub, Publisher: NewPublisher(hub), closeStore: store.Close}
}

type recordingFanout struct {
	mu      sync.Mutex
	events  []sessionstream.UIEvent
	signals chan struct{}
}

func newRecordingFanout() *recordingFanout {
	return &recordingFanout{signals: make(chan struct{}, 16)}
}

func (f *recordingFanout) PublishUI(_ context.Context, _ sessionstream.SessionId, _ uint64, events []sessionstream.UIEvent) error {
	f.mu.Lock()
	f.events = append(f.events, events...)
	f.mu.Unlock()
	select {
	case f.signals <- struct{}{}:
	default:
	}
	return nil
}

func (f *recordingFanout) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events)
}

package runtimeevents

import (
	"context"
	"testing"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/stretchr/testify/require"
)

func TestOpenPublisherSubscriberGoChannel(t *testing.T) {
	resources, err := OpenPublisherSubscriber(Config{Backend: BackendGoChannel})
	require.NoError(t, err)
	defer func() { require.NoError(t, resources.Close()) }()

	publisher := resources.EventPublisher()
	subscriber := resources.EventSubscriber()
	require.NotNil(t, publisher)
	require.NotNil(t, subscriber)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	messages, err := subscriber.Subscribe(ctx)
	require.NoError(t, err)

	err = publisher.Publish(&runtimev1.RuntimeEventV1{
		Source:  runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_WORKER,
		Message: "hello",
	})
	require.NoError(t, err)

	msg := <-messages
	event, err := EventFromMessage(msg)
	require.NoError(t, err)
	require.Equal(t, "hello", event.Message)
}

func TestOpenPublisherOffBackendIsNoop(t *testing.T) {
	resources, err := OpenPublisher(Config{})
	require.NoError(t, err)
	defer func() { require.NoError(t, resources.Close()) }()

	err = resources.EventPublisher().Publish(&runtimev1.RuntimeEventV1{
		Source:  runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_SERVER,
		Message: "ignored",
	})
	require.NoError(t, err)
}

func TestConfigValidateRejectsUnknownBackend(t *testing.T) {
	err := Config{Backend: Backend("bogus")}.Validate()
	require.Error(t, err)
}

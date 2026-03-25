package runtimeevents

import (
	"context"
	"testing"
	"time"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/stretchr/testify/require"
)

func TestMessageFromEventSetsMetadata(t *testing.T) {
	event := buildTestEvent(t)

	msg, err := MessageFromEvent(event)
	require.NoError(t, err)
	require.Equal(t, event.Id, msg.UUID)
	require.Equal(t, ContentTypeProtobuf, msg.Metadata.Get(MetadataContentType))
	require.Equal(t, "1", msg.Metadata.Get(MetadataSchemaVersion))
	require.Equal(t, event.Kind.String(), msg.Metadata.Get(MetadataEventKind))
	require.Equal(t, event.Source.String(), msg.Metadata.Get(MetadataEventSource))
	require.Equal(t, event.WorkflowId, msg.Metadata.Get(MetadataWorkflowID))
}

func TestPublisherSubscriberRoundTripWithGoChannel(t *testing.T) {
	pubsub := NewGoChannelPubSub()
	publisher := NewPublisher(pubsub, "")
	subscriber := NewSubscriber(pubsub, "")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages, err := subscriber.Subscribe(ctx)
	require.NoError(t, err)

	expected := buildTestEvent(t)
	require.NoError(t, publisher.Publish(expected))

	select {
	case msg := <-messages:
		require.NotNil(t, msg)

		event, err := EventFromMessage(msg)
		require.NoError(t, err)
		require.Equal(t, expected.Id, event.Id)
		require.Equal(t, expected.Kind, event.Kind)
		require.Equal(t, expected.Message, event.Message)
		require.True(t, msg.Ack())
	case <-ctx.Done():
		t.Fatal("timed out waiting for runtime event message")
	}
}

func TestMessageFromEventAssignsUUIDWhenMissing(t *testing.T) {
	event := &runtimev1.RuntimeEventV1{
		Kind:     runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_LOG_LINE,
		Severity: runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_INFO,
	}

	msg, err := MessageFromEvent(event)
	require.NoError(t, err)
	require.NotEmpty(t, msg.UUID)
	require.Equal(t, msg.UUID, event.Id)
}

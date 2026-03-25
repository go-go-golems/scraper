package runtimeevents

import (
	"context"
	"strconv"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
)

const (
	TopicRuntimeEventsV1  = "scraper.runtime.v1.events"
	ContentTypeProtobuf   = "application/x-protobuf; message=scraper.runtime.v1.RuntimeEventV1"
	MetadataContentType   = "content_type"
	MetadataSchemaVersion = "schema_version"
	MetadataEventKind     = "event_kind"
	MetadataEventSource   = "event_source"
	MetadataOccurredAt    = "occurred_at"
	MetadataComponent     = "component"
	MetadataWorkflowID    = "workflow_id"
	MetadataOpID          = "op_id"
	MetadataSite          = "site"
	MetadataQueue         = "queue"
	MetadataWorkerID      = "worker_id"
	MetadataRequestID     = "request_id"
	MetadataArtifactID    = "artifact_id"
)

type Publisher struct {
	topic     string
	publisher message.Publisher
}

type Subscriber struct {
	topic      string
	subscriber message.Subscriber
}

func NewPublisher(publisher message.Publisher, topic string) *Publisher {
	if topic == "" {
		topic = TopicRuntimeEventsV1
	}
	return &Publisher{
		topic:     topic,
		publisher: publisher,
	}
}

func NewSubscriber(subscriber message.Subscriber, topic string) *Subscriber {
	if topic == "" {
		topic = TopicRuntimeEventsV1
	}
	return &Subscriber{
		topic:      topic,
		subscriber: subscriber,
	}
}

func NewGoChannelPubSub() *gochannel.GoChannel {
	return gochannel.NewGoChannel(gochannel.Config{}, watermill.NopLogger{})
}

func (p *Publisher) Publish(event *runtimev1.RuntimeEventV1) error {
	msg, err := MessageFromEvent(event)
	if err != nil {
		return err
	}
	return p.publisher.Publish(p.topic, msg)
}

func (s *Subscriber) Subscribe(ctx context.Context) (<-chan *message.Message, error) {
	return s.subscriber.Subscribe(ctx, s.topic)
}

func MessageFromEvent(event *runtimev1.RuntimeEventV1) (*message.Message, error) {
	event, err := Normalize(event)
	if err != nil {
		return nil, err
	}

	payload, err := MarshalBinary(event)
	if err != nil {
		return nil, err
	}

	id := event.Id
	if id == "" {
		id = watermill.NewUUID()
		event.Id = id
	}

	msg := message.NewMessage(id, payload)
	msg.Metadata.Set(MetadataContentType, ContentTypeProtobuf)
	msg.Metadata.Set(MetadataSchemaVersion, strconv.FormatUint(uint64(event.SchemaVersion), 10))
	msg.Metadata.Set(MetadataEventKind, event.Kind.String())
	msg.Metadata.Set(MetadataEventSource, event.Source.String())

	setMetadataIfPresent(msg, MetadataComponent, event.Component)
	setMetadataIfPresent(msg, MetadataWorkflowID, event.WorkflowId)
	setMetadataIfPresent(msg, MetadataOpID, event.OpId)
	setMetadataIfPresent(msg, MetadataSite, event.Site)
	setMetadataIfPresent(msg, MetadataQueue, event.Queue)
	setMetadataIfPresent(msg, MetadataWorkerID, event.WorkerId)
	setMetadataIfPresent(msg, MetadataRequestID, event.RequestId)
	setMetadataIfPresent(msg, MetadataArtifactID, event.ArtifactId)

	if event.OccurredAt != nil {
		msg.Metadata.Set(MetadataOccurredAt, event.OccurredAt.AsTime().UTC().Format(time.RFC3339Nano))
	}

	return msg, nil
}

func EventFromMessage(msg *message.Message) (*runtimev1.RuntimeEventV1, error) {
	event, err := UnmarshalBinary(msg.Payload)
	if err != nil {
		return nil, err
	}
	if event.Id == "" {
		event.Id = msg.UUID
	}
	return event, nil
}

func setMetadataIfPresent(msg *message.Message, key, value string) {
	if value == "" {
		return
	}
	msg.Metadata.Set(key, value)
}

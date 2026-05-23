package runtimestream

import (
	"context"
	"errors"
	"fmt"
	"strings"

	streamv1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/sessionstream/v1"
	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Publisher struct {
	hub *sessionstream.Hub
}

func NewPublisher(hub *sessionstream.Hub) *Publisher {
	if hub == nil {
		return nil
	}
	return &Publisher{hub: hub}
}

func (p *Publisher) Publish(ctx context.Context, event *runtimev1.RuntimeEventV1) error {
	if p == nil || p.hub == nil || event == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := NormalizeRuntimeEvent(event)
	ids := RuntimeEventSessionIDs(normalized)
	errs := make([]error, 0, len(ids))
	for _, sid := range ids {
		payload := &streamv1.PublishRuntimeEventCommand{Event: proto.Clone(normalized).(*runtimev1.RuntimeEventV1)}
		if err := p.hub.Submit(ctx, sid, CommandPublishRuntimeEvent, payload); err != nil {
			errs = append(errs, fmt.Errorf("session %s: %w", sid, err))
		}
	}
	return errors.Join(errs...)
}

func NormalizeRuntimeEvent(event *runtimev1.RuntimeEventV1) *runtimev1.RuntimeEventV1 {
	if event == nil {
		return nil
	}
	cloned := proto.Clone(event).(*runtimev1.RuntimeEventV1)
	_, _ = runtimeevents.Normalize(cloned)
	if strings.TrimSpace(cloned.Id) == "" {
		cloned.Id = uuid.NewString()
	}
	if cloned.OccurredAt == nil {
		cloned.OccurredAt = timestamppb.Now()
	}
	return cloned
}

func RegisterCommands(hub *sessionstream.Hub) error {
	if hub == nil {
		return fmt.Errorf("hub is nil")
	}
	return hub.RegisterCommand(CommandPublishRuntimeEvent, handlePublishRuntimeEvent)
}

func handlePublishRuntimeEvent(ctx context.Context, cmd sessionstream.Command, _ *sessionstream.Session, pub sessionstream.EventPublisher) error {
	payload, ok := cmd.Payload.(*streamv1.PublishRuntimeEventCommand)
	if !ok || payload.GetEvent() == nil {
		return fmt.Errorf("publish runtime event payload must be %T, got %T", &streamv1.PublishRuntimeEventCommand{}, cmd.Payload)
	}
	normalized := NormalizeRuntimeEvent(payload.GetEvent())
	return pub.Publish(ctx, sessionstream.Event{
		Name:      EventRuntimeEventObserved,
		SessionId: cmd.SessionId,
		Payload:   &streamv1.RuntimeEventObserved{Event: normalized},
	})
}

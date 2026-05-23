package runtimestream

import (
	"context"
	"fmt"
	"sort"
	"strings"

	streamv1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/sessionstream/v1"
	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"google.golang.org/protobuf/proto"
)

type UIProjection struct{}

func (UIProjection) Project(_ context.Context, ev sessionstream.Event, _ *sessionstream.Session, _ sessionstream.TimelineView) ([]sessionstream.UIEvent, error) {
	if ev.Name != EventRuntimeEventObserved {
		return nil, nil
	}
	observed, ok := ev.Payload.(*streamv1.RuntimeEventObserved)
	if !ok || observed.GetEvent() == nil {
		return nil, fmt.Errorf("runtime event projection payload must be %T, got %T", &streamv1.RuntimeEventObserved{}, ev.Payload)
	}
	return []sessionstream.UIEvent{{
		Name: UIEventRuntimeEventAppended,
		Payload: &streamv1.RuntimeEventAppended{
			Event: proto.Clone(observed.GetEvent()).(*runtimev1.RuntimeEventV1),
		},
	}}, nil
}

type TimelineProjection struct {
	MaxEntitiesPerSession int
}

func (p TimelineProjection) Project(_ context.Context, ev sessionstream.Event, _ *sessionstream.Session, view sessionstream.TimelineView) ([]sessionstream.TimelineEntity, error) {
	if ev.Name != EventRuntimeEventObserved {
		return nil, nil
	}
	observed, ok := ev.Payload.(*streamv1.RuntimeEventObserved)
	if !ok || observed.GetEvent() == nil {
		return nil, fmt.Errorf("runtime event timeline payload must be %T, got %T", &streamv1.RuntimeEventObserved{}, ev.Payload)
	}
	event := observed.GetEvent()
	id := strings.TrimSpace(event.GetId())
	if id == "" {
		id = fmt.Sprintf("ordinal-%020d", ev.Ordinal)
	}

	entities := []sessionstream.TimelineEntity{{
		Kind:             EntityRuntimeEvent,
		Id:               id,
		CreatedOrdinal:   ev.Ordinal,
		LastEventOrdinal: ev.Ordinal,
		Payload: &streamv1.RuntimeEventEntity{
			Event: proto.Clone(event).(*runtimev1.RuntimeEventV1),
		},
	}}
	entities = append(entities, p.tombstonesForRetention(view, id, ev.Ordinal)...)
	return entities, nil
}

func (p TimelineProjection) tombstonesForRetention(view sessionstream.TimelineView, newID string, ord uint64) []sessionstream.TimelineEntity {
	limit := p.MaxEntitiesPerSession
	if limit <= 0 || view == nil {
		return nil
	}
	existing := view.List(EntityRuntimeEvent)
	if len(existing)+1 <= limit {
		return nil
	}
	candidates := make([]sessionstream.TimelineEntity, 0, len(existing))
	for _, entity := range existing {
		if entity.Id == newID {
			continue
		}
		candidates = append(candidates, entity)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].LastEventOrdinal == candidates[j].LastEventOrdinal {
			return candidates[i].Id < candidates[j].Id
		}
		return candidates[i].LastEventOrdinal < candidates[j].LastEventOrdinal
	})
	remove := len(existing) + 1 - limit
	if remove > len(candidates) {
		remove = len(candidates)
	}
	out := make([]sessionstream.TimelineEntity, 0, remove)
	for _, entity := range candidates[:remove] {
		out = append(out, sessionstream.TimelineEntity{
			Kind:             EntityRuntimeEvent,
			Id:               entity.Id,
			CreatedOrdinal:   entity.CreatedOrdinal,
			LastEventOrdinal: ord,
			Tombstone:        true,
		})
	}
	return out
}

func DefaultTimelineProjection() TimelineProjection {
	return TimelineProjection{MaxEntitiesPerSession: runtimeevents.DefaultRecentEventLimit}
}

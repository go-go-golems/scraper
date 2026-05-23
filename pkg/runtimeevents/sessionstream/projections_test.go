package runtimestream

import (
	"context"
	"testing"

	streamv1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/sessionstream/v1"
	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
	"github.com/stretchr/testify/require"
)

func TestRegisterSchemas(t *testing.T) {
	reg := sessionstream.NewSchemaRegistry()
	require.NoError(t, RegisterSchemas(reg))
	_, ok := reg.CommandSchema(CommandPublishRuntimeEvent)
	require.True(t, ok)
	_, ok = reg.EventSchema(EventRuntimeEventObserved)
	require.True(t, ok)
	_, ok = reg.UIEventSchema(UIEventRuntimeEventAppended)
	require.True(t, ok)
	_, ok = reg.TimelineEntitySchema(EntityRuntimeEvent)
	require.True(t, ok)
}

func TestUIProjectionWrapsRuntimeEvent(t *testing.T) {
	event := &runtimev1.RuntimeEventV1{Id: "ev-1", Message: "hello"}
	uiEvents, err := (UIProjection{}).Project(context.Background(), sessionstream.Event{
		Name:    EventRuntimeEventObserved,
		Payload: &streamv1.RuntimeEventObserved{Event: event},
	}, nil, nil)
	require.NoError(t, err)
	require.Len(t, uiEvents, 1)
	require.Equal(t, UIEventRuntimeEventAppended, uiEvents[0].Name)
	payload, ok := uiEvents[0].Payload.(*streamv1.RuntimeEventAppended)
	require.True(t, ok)
	require.Equal(t, "ev-1", payload.GetEvent().GetId())
}

func TestTimelineProjectionCreatesEntityAndTombstonesOldEvents(t *testing.T) {
	view := staticTimelineView{entities: []sessionstream.TimelineEntity{
		{Kind: EntityRuntimeEvent, Id: "old-1", LastEventOrdinal: 1},
		{Kind: EntityRuntimeEvent, Id: "old-2", LastEventOrdinal: 2},
	}}
	entities, err := (TimelineProjection{MaxEntitiesPerSession: 2}).Project(context.Background(), sessionstream.Event{
		Name:    EventRuntimeEventObserved,
		Ordinal: 3,
		Payload: &streamv1.RuntimeEventObserved{Event: &runtimev1.RuntimeEventV1{Id: "new-1"}},
	}, nil, view)
	require.NoError(t, err)
	require.Len(t, entities, 2)
	require.Equal(t, EntityRuntimeEvent, entities[0].Kind)
	require.Equal(t, "new-1", entities[0].Id)
	payload, ok := entities[0].Payload.(*streamv1.RuntimeEventEntity)
	require.True(t, ok)
	require.Equal(t, "new-1", payload.GetEvent().GetId())
	require.Equal(t, "old-1", entities[1].Id)
	require.True(t, entities[1].Tombstone)
}

type staticTimelineView struct {
	entities []sessionstream.TimelineEntity
}

func (v staticTimelineView) Get(kind, id string) (sessionstream.TimelineEntity, bool) {
	for _, entity := range v.entities {
		if entity.Kind == kind && entity.Id == id {
			return entity, true
		}
	}
	return sessionstream.TimelineEntity{}, false
}

func (v staticTimelineView) List(kind string) []sessionstream.TimelineEntity {
	out := make([]sessionstream.TimelineEntity, 0, len(v.entities))
	for _, entity := range v.entities {
		if entity.Kind == kind {
			out = append(out, entity)
		}
	}
	return out
}

func (v staticTimelineView) Ordinal() uint64 { return 0 }

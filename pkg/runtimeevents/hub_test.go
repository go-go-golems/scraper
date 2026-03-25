package runtimeevents

import (
	"context"
	"testing"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/stretchr/testify/require"
)

func TestHubRecentFiltersAndLimits(t *testing.T) {
	hub := NewHub(4)
	hub.Add(&runtimev1.RuntimeEventV1{Id: "1", WorkflowId: "wf-1", Message: "one"})
	hub.Add(&runtimev1.RuntimeEventV1{Id: "2", WorkflowId: "wf-2", Message: "two"})
	hub.Add(&runtimev1.RuntimeEventV1{Id: "3", WorkflowId: "wf-1", Message: "three"})

	events := hub.Recent(Filter{WorkflowID: "wf-1", Limit: 1})
	require.Len(t, events, 1)
	require.Equal(t, "3", events[0].Id)
}

func TestHubSubscribeReceivesMatchingEvents(t *testing.T) {
	hub := NewHub(8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := hub.Subscribe(ctx, Filter{WorkflowID: "wf-9"})
	hub.Add(&runtimev1.RuntimeEventV1{Id: "1", WorkflowId: "wf-1"})
	hub.Add(&runtimev1.RuntimeEventV1{Id: "2", WorkflowId: "wf-9"})

	event := <-stream
	require.Equal(t, "2", event.Id)
}

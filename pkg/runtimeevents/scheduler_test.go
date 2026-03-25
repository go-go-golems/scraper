package runtimeevents

import (
	"testing"
	"time"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	"github.com/stretchr/testify/require"
)

func TestFromSchedulerEventMapsFailureDetails(t *testing.T) {
	now := time.Unix(1_710_000_100, 0).UTC()

	event, err := FromSchedulerEvent(scheduler.Event{
		Kind:       scheduler.EventOpFailed,
		OccurredAt: now,
		WorkflowID: "wf-1",
		OpID:       "op-1",
		Site:       "hackernews",
		Queue:      "site:hn:http",
		Attempt:    2,
		Message:    "op failed",
		Error: &model.OpError{
			Code:      "runner_error",
			Message:   "boom",
			Retryable: false,
		},
	}, "scheduler", "worker-1")
	require.NoError(t, err)

	require.Equal(t, runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_SCHEDULER, event.Source)
	require.Equal(t, runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_OP_FAILED, event.Kind)
	require.Equal(t, runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_ERROR, event.Severity)
	require.Equal(t, "wf-1", event.WorkflowId)
	require.Equal(t, "op-1", event.OpId)
	require.Equal(t, "worker-1", event.WorkerId)
	require.Equal(t, "runner_error", event.Payload.GetFields()["errorCode"].GetStringValue())
	require.Equal(t, "boom", event.Payload.GetFields()["errorMessage"].GetStringValue())
	require.False(t, event.Payload.GetFields()["retryable"].GetBoolValue())
	require.Equal(t, float64(2), event.Payload.GetFields()["attempt"].GetNumberValue())
	require.Equal(t, now, event.OccurredAt.AsTime())
}

func TestFromSchedulerEventMapsIdleEvent(t *testing.T) {
	event, err := FromSchedulerEvent(scheduler.Event{
		Kind:    scheduler.EventIdle,
		Message: "no leaseable queues",
	}, "scheduler", "worker-1")
	require.NoError(t, err)

	require.Equal(t, runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_WORKER_IDLE, event.Kind)
	require.Equal(t, runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_DEBUG, event.Severity)
	require.Nil(t, event.Payload)
}

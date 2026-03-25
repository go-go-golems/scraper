package runtimeevents

import (
	"fmt"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func FromSchedulerEvent(event scheduler.Event, component string, workerID string) (*runtimev1.RuntimeEventV1, error) {
	payload, err := schedulerPayload(event)
	if err != nil {
		return nil, err
	}

	result := &runtimev1.RuntimeEventV1{
		SchemaVersion: SchemaVersionV1,
		Source:        runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_SCHEDULER,
		Component:     component,
		Kind:          schedulerEventKind(event.Kind),
		Severity:      schedulerEventSeverity(event.Kind),
		Message:       event.Message,
		WorkflowId:    string(event.WorkflowID),
		OpId:          string(event.OpID),
		Site:          string(event.Site),
		Queue:         string(event.Queue),
		WorkerId:      workerID,
		Payload:       payload,
	}
	if !event.OccurredAt.IsZero() {
		result.OccurredAt = timestamppb.New(event.OccurredAt.UTC())
	}

	return result, nil
}

func schedulerEventKind(kind scheduler.EventKind) runtimev1.RuntimeEventKind {
	switch kind {
	case scheduler.EventWorkflowCreated:
		return runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_WORKFLOW_CREATED
	case scheduler.EventWorkflowUpdated:
		return runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_WORKFLOW_UPDATED
	case scheduler.EventOpLeased:
		return runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_OP_LEASED
	case scheduler.EventOpSucceeded:
		return runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_OP_SUCCEEDED
	case scheduler.EventOpRetried:
		return runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_OP_RETRIED
	case scheduler.EventOpFailed:
		return runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_OP_FAILED
	case scheduler.EventQueueRateLimited:
		return runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_QUEUE_RATE_LIMITED
	case scheduler.EventIdle:
		return runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_WORKER_IDLE
	default:
		return runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_UNSPECIFIED
	}
}

func schedulerEventSeverity(kind scheduler.EventKind) runtimev1.RuntimeEventSeverity {
	switch kind {
	case scheduler.EventOpFailed:
		return runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_ERROR
	case scheduler.EventOpRetried, scheduler.EventQueueRateLimited:
		return runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_WARN
	case scheduler.EventIdle:
		return runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_DEBUG
	default:
		return runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_INFO
	}
}

func schedulerPayload(event scheduler.Event) (*structpb.Struct, error) {
	data := map[string]any{}
	if event.Attempt > 0 {
		data["attempt"] = float64(event.Attempt)
	}
	if event.Status != "" {
		data["workflowStatus"] = string(event.Status)
	}
	if event.Error != nil {
		data["errorCode"] = event.Error.Code
		data["errorMessage"] = event.Error.Message
		data["retryable"] = event.Error.Retryable
	}
	if len(data) == 0 {
		return nil, nil
	}

	payload, err := structpb.NewStruct(data)
	if err != nil {
		return nil, fmt.Errorf("build scheduler payload: %w", err)
	}
	return payload, nil
}

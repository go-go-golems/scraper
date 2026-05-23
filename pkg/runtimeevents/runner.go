package runtimeevents

import (
	"context"
	"fmt"
	"time"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ObservedRunner struct {
	base      runner.Runner
	publisher Publisher
	component string
	workerID  string
}

func WrapRunner(base runner.Runner, publisher Publisher, component string, workerID string) runner.Runner {
	if base == nil || publisher == nil {
		return base
	}
	return &ObservedRunner{
		base:      base,
		publisher: publisher,
		component: component,
		workerID:  workerID,
	}
}

func (r *ObservedRunner) Kind() string {
	return r.base.Kind()
}

func (r *ObservedRunner) Run(ctx context.Context, runCtx runner.RunContext) (*model.OpResult, error) {
	startedAt := time.Now().UTC()
	_ = r.publisher.Publish(ctx, buildRunnerLogEvent(
		runCtx,
		r.component,
		r.workerID,
		runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_INFO,
		"runner started",
		startedAt,
		map[string]any{
			"runnerKind": r.base.Kind(),
		},
		"",
	))

	result, err := r.base.Run(ctx, runCtx)
	completedAt := time.Now().UTC()

	payload := map[string]any{
		"runnerKind":        r.base.Kind(),
		"durationMillis":    completedAt.Sub(startedAt).Milliseconds(),
		"emittedCount":      0,
		"artifactCount":     0,
		"recordWriteCount":  0,
		"artifactSummaries": []any{},
	}
	severity := runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_INFO
	message := "runner completed"
	errorCode := ""
	if err != nil {
		severity = runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_ERROR
		message = "runner failed"
		errorCode = "runner_error"
		payload["error"] = err.Error()
	} else if result != nil {
		payload["emittedCount"] = len(result.Emitted)
		payload["artifactCount"] = len(result.Artifacts)
		payload["recordWriteCount"] = len(result.Records)
		payload["artifactSummaries"] = artifactSummaries(result.Artifacts)
		if result.Error != nil {
			severity = runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_ERROR
			message = "runner returned op error"
			errorCode = result.Error.Code
			payload["error"] = result.Error.Message
			payload["retryable"] = result.Error.Retryable
		}
	}

	_ = r.publisher.Publish(ctx, buildRunnerLogEvent(
		runCtx,
		r.component,
		r.workerID,
		severity,
		message,
		completedAt,
		payload,
		errorCode,
	))

	return result, err
}

func buildRunnerLogEvent(
	runCtx runner.RunContext,
	component string,
	workerID string,
	severity runtimev1.RuntimeEventSeverity,
	message string,
	occurredAt time.Time,
	payload map[string]any,
	errorCode string,
) *runtimev1.RuntimeEventV1 {
	event := &runtimev1.RuntimeEventV1{
		SchemaVersion: SchemaVersionV1,
		Source:        runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_RUNNER,
		Component:     component,
		Kind:          runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_LOG_LINE,
		Severity:      severity,
		Message:       message,
		WorkflowId:    string(runCtx.Workflow.ID),
		OpId:          string(runCtx.Op.ID),
		Site:          string(runCtx.Op.Site),
		Queue:         string(runCtx.Op.Queue),
		WorkerId:      workerID,
	}
	if !occurredAt.IsZero() {
		event.OccurredAt = timestamppb.New(occurredAt)
	}
	if len(payload) > 0 {
		if errorCode != "" {
			payload["errorCode"] = errorCode
		}
		if structPayload, err := structpb.NewStruct(payload); err == nil {
			event.Payload = structPayload
		}
	}
	return event
}

func artifactSummaries(artifacts []model.ArtifactWrite) []any {
	if len(artifacts) == 0 {
		return []any{}
	}
	summaries := make([]any, 0, len(artifacts))
	for _, artifact := range artifacts {
		summaries = append(summaries, map[string]any{
			"id":          string(artifact.ID),
			"name":        artifact.Name,
			"kind":        artifact.Kind,
			"contentType": artifact.ContentType,
			"sizeBytes":   len(artifact.Body),
		})
	}
	return summaries
}

func EmitSimpleEvent(
	ctx context.Context,
	publisher Publisher,
	event *runtimev1.RuntimeEventV1,
) error {
	if publisher == nil || event == nil {
		return nil
	}
	if event.OccurredAt == nil {
		event.OccurredAt = timestamppb.New(time.Now().UTC())
	}
	if event.SchemaVersion == 0 {
		event.SchemaVersion = SchemaVersionV1
	}
	return publisher.Publish(ctx, event)
}

func BuildPayload(fields map[string]any) *structpb.Struct {
	if len(fields) == 0 {
		return nil
	}
	payload, err := structpb.NewStruct(fields)
	if err != nil {
		return nil
	}
	return payload
}

func BuildServerEvent(
	source runtimev1.RuntimeEventSource,
	kind runtimev1.RuntimeEventKind,
	severity runtimev1.RuntimeEventSeverity,
	component string,
	message string,
	fields map[string]any,
) *runtimev1.RuntimeEventV1 {
	event := &runtimev1.RuntimeEventV1{
		SchemaVersion: SchemaVersionV1,
		Source:        source,
		Component:     component,
		Kind:          kind,
		Severity:      severity,
		Message:       message,
		OccurredAt:    timestamppb.New(time.Now().UTC()),
		Payload:       BuildPayload(fields),
	}
	if workflowID, ok := fields["workflowId"].(string); ok {
		event.WorkflowId = workflowID
	}
	if opID, ok := fields["opId"].(string); ok {
		event.OpId = opID
	}
	if site, ok := fields["site"].(string); ok {
		event.Site = site
	}
	if workerID, ok := fields["workerId"].(string); ok {
		event.WorkerId = workerID
	}
	if requestID, ok := fields["requestId"].(string); ok {
		event.RequestId = requestID
	}
	if artifactID, ok := fields["artifactId"].(string); ok {
		event.ArtifactId = artifactID
	}
	return event
}

func RequireResources(resources *Resources) error {
	if resources == nil {
		return fmt.Errorf("runtime event resources are nil")
	}
	return nil
}

package runtimeevents

import (
	"testing"
	"time"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNormalizeSetsSchemaVersion(t *testing.T) {
	event, err := Normalize(&runtimev1.RuntimeEventV1{})
	require.NoError(t, err)
	require.Equal(t, uint32(SchemaVersionV1), event.SchemaVersion)
}

func TestMarshalBinaryRoundTrip(t *testing.T) {
	event := buildTestEvent(t)

	data, err := MarshalBinary(event)
	require.NoError(t, err)

	decoded, err := UnmarshalBinary(data)
	require.NoError(t, err)
	require.Equal(t, event.Id, decoded.Id)
	require.Equal(t, event.Kind, decoded.Kind)
	require.Equal(t, uint32(SchemaVersionV1), decoded.SchemaVersion)
	require.Equal(t, "ok", decoded.Labels["status"])
}

func TestMarshalJSONRoundTrip(t *testing.T) {
	event := buildTestEvent(t)

	data, err := MarshalJSON(event)
	require.NoError(t, err)
	require.Contains(t, string(data), `"schemaVersion":1`)
	require.Contains(t, string(data), `"workflowId":"wf-123"`)

	decoded, err := UnmarshalJSON(data)
	require.NoError(t, err)
	require.Equal(t, event.Id, decoded.Id)
	require.Equal(t, event.Message, decoded.Message)
	require.Equal(t, "ok", decoded.Labels["status"])
	require.Equal(t, "https://example.com", decoded.Payload.GetFields()["url"].GetStringValue())
}

func buildTestEvent(t *testing.T) *runtimev1.RuntimeEventV1 {
	t.Helper()

	payload, err := structpb.NewStruct(map[string]any{
		"url":   "https://example.com",
		"count": 2,
	})
	require.NoError(t, err)

	return &runtimev1.RuntimeEventV1{
		Id:            "evt-123",
		Source:        runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_WORKER,
		Component:     "worker",
		Kind:          runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_OP_SUCCEEDED,
		Severity:      runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_INFO,
		OccurredAt:    timestamppb.New(time.Unix(1_710_000_000, 0).UTC()),
		Message:       "op succeeded",
		WorkflowId:    "wf-123",
		OpId:          "op-456",
		Site:          "hackernews",
		Queue:         "site:hn:http",
		WorkerId:      "worker-1",
		RequestId:     "req-1",
		ArtifactId:    "art-1",
		Labels:        map[string]string{"status": "ok"},
		Payload:       payload,
		SchemaVersion: SchemaVersionV1,
	}
}

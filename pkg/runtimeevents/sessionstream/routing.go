package runtimestream

import (
	"strings"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
)

func RuntimeEventSessionIDs(event *runtimev1.RuntimeEventV1) []sessionstream.SessionId {
	ids := []sessionstream.SessionId{SessionRuntimeGlobal}
	if event == nil {
		return ids
	}
	if workflowID := strings.TrimSpace(event.GetWorkflowId()); workflowID != "" {
		ids = append(ids, WorkflowSessionID(workflowID))
	}
	return dedupeSessionIDs(ids)
}

func dedupeSessionIDs(ids []sessionstream.SessionId) []sessionstream.SessionId {
	seen := map[sessionstream.SessionId]struct{}{}
	out := make([]sessionstream.SessionId, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

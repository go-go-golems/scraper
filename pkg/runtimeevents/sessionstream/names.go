package runtimestream

import sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"

const (
	// TopicRuntimeEventsSessionstreamV1 is the Watermill topic carrying
	// sessionstream runtime-event envelopes for scraper.
	TopicRuntimeEventsSessionstreamV1 = "scraper.runtime.sessionstream.v1.events"

	CommandPublishRuntimeEvent  = "scraper.runtime.PublishRuntimeEvent"
	EventRuntimeEventObserved   = "scraper.runtime.RuntimeEventObserved"
	UIEventRuntimeEventAppended = "scraper.runtime.RuntimeEventAppended"
	EntityRuntimeEvent          = "scraper.runtime.RuntimeEvent"

	SessionRuntimeGlobal sessionstream.SessionId = "runtime:global"
)

func WorkflowSessionID(workflowID string) sessionstream.SessionId {
	if workflowID == "" {
		return ""
	}
	return sessionstream.SessionId("workflow:" + workflowID)
}

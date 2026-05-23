package runtimestream

import (
	"fmt"

	streamv1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/sessionstream/v1"
	sessionstream "github.com/go-go-golems/sessionstream/pkg/sessionstream"
)

func RegisterSchemas(reg *sessionstream.SchemaRegistry) error {
	if reg == nil {
		return fmt.Errorf("schema registry is nil")
	}
	for _, err := range []error{
		reg.RegisterCommand(CommandPublishRuntimeEvent, &streamv1.PublishRuntimeEventCommand{}),
		reg.RegisterEvent(EventRuntimeEventObserved, &streamv1.RuntimeEventObserved{}),
		reg.RegisterUIEvent(UIEventRuntimeEventAppended, &streamv1.RuntimeEventAppended{}),
		reg.RegisterTimelineEntity(EntityRuntimeEvent, &streamv1.RuntimeEventEntity{}),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

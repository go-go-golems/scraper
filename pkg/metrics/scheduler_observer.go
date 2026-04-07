package metrics

import (
	"context"

	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
)

func NewSchedulerObserver(registry *Registry) scheduler.Observer {
	if registry == nil {
		return nil
	}

	return scheduler.ObserverFunc(func(ctx context.Context, event scheduler.Event) {
		switch event.Kind {
		case scheduler.EventOpLeased:
			registry.ObserveOpLeased(event.Site, event.Queue, event.RunnerKind)
		case scheduler.EventOpFailed:
			errorCode := ""
			if event.Error != nil {
				errorCode = event.Error.Code
			}
			registry.ObserveOpFailed(event.Site, event.Queue, event.RunnerKind, errorCode)
		case scheduler.EventOpRetried:
			registry.ObserveOpRetried(event.Site, event.Queue, event.RunnerKind)
		case scheduler.EventQueueRateLimited:
			registry.ObserveQueueRateLimited(event.Site, event.Queue)
		}
	})
}

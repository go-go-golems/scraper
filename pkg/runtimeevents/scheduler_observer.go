package runtimeevents

import (
	"context"

	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	"github.com/rs/zerolog/log"
)

func NewSchedulerObserver(publisher Publisher, component string, workerID string) scheduler.Observer {
	if publisher == nil {
		return nil
	}

	return scheduler.ObserverFunc(func(ctx context.Context, event scheduler.Event) {
		runtimeEvent, err := FromSchedulerEvent(event, component, workerID)
		if err != nil {
			log.Warn().Err(err).Str("component", component).Str("worker_id", workerID).Msg("failed to map scheduler event")
			return
		}
		if runtimeEvent == nil {
			return
		}
		if err := publisher.Publish(ctx, runtimeEvent); err != nil {
			log.Warn().Err(err).Str("component", component).Str("worker_id", workerID).Msg("failed to publish scheduler event")
		}
	})
}

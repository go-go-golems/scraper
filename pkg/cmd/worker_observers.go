package cmd

import (
	"context"

	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	"github.com/go-go-golems/scraper/pkg/metrics"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
)

func newWorkerObserver(eventPublisher runtimeevents.Publisher, metricsRegistry *metrics.Registry, workerID string) scheduler.Observer {
	return composeSchedulerObservers(
		runtimeevents.NewSchedulerObserver(eventPublisher, "worker-scheduler", workerID),
		metrics.NewSchedulerObserver(metricsRegistry),
	)
}

func composeSchedulerObservers(observers ...scheduler.Observer) scheduler.Observer {
	filtered := make([]scheduler.Observer, 0, len(observers))
	for _, observer := range observers {
		if observer != nil {
			filtered = append(filtered, observer)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return scheduler.ObserverFunc(func(ctx context.Context, event scheduler.Event) {
		for _, observer := range filtered {
			observer.OnSchedulerEvent(ctx, event)
		}
	})
}

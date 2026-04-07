package server

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	"github.com/rs/zerolog/log"
)

func startRuntimeEventRouter(resources *runtimeevents.Resources, hub *runtimeevents.Hub) (*message.Router, error) {
	if resources == nil || resources.Subscriber == nil {
		return nil, nil
	}

	router, err := message.NewRouter(message.RouterConfig{}, watermill.NopLogger{})
	if err != nil {
		return nil, err
	}
	router.AddNoPublisherHandler("runtime-events-hub", resources.Topic, resources.Subscriber, func(msg *message.Message) error {
		event, err := runtimeevents.EventFromMessage(msg)
		if err != nil {
			return err
		}
		hub.Add(event)
		return nil
	})

	go func() {
		if err := router.Run(context.Background()); err != nil {
			log.Warn().Err(err).Msg("runtime event router stopped")
		}
	}()
	return router, nil
}

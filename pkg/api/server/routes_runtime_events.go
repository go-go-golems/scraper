package server

import (
	"net/http"

	"github.com/go-go-golems/scraper/pkg/api/handlers"
)

func registerRuntimeEventRoutes(mux *http.ServeMux, runtimeEventsHandler *handlers.RuntimeEventsHandler) {
	mux.HandleFunc("GET /api/v1/runtime-events", runtimeEventsHandler.List)
	mux.HandleFunc("GET /api/v1/runtime-events/stream", runtimeEventsHandler.Stream)
}

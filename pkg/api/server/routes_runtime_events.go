package server

import (
	"net/http"

	ws "github.com/go-go-golems/sessionstream/pkg/sessionstream/transport/ws"
)

func registerRuntimeEventRoutes(mux *http.ServeMux, runtimeEventsWS *ws.Server) {
	if runtimeEventsWS == nil {
		return
	}
	mux.Handle("GET /api/v1/runtime-events/ws", runtimeEventsWS)
}

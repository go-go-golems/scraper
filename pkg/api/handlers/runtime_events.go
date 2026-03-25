package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-go-golems/scraper/pkg/runtimeevents"
)

type RuntimeEventsHandler struct {
	hub *runtimeevents.Hub
}

func NewRuntimeEventsHandler(hub *runtimeevents.Hub) *RuntimeEventsHandler {
	return &RuntimeEventsHandler{hub: hub}
}

func (h *RuntimeEventsHandler) List(w http.ResponseWriter, r *http.Request) {
	events := h.hub.Recent(runtimeEventFilterFromRequest(r))
	rawEvents := make([]json.RawMessage, 0, len(events))
	for _, event := range events {
		raw, err := runtimeevents.MarshalJSON(event)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "event_encode_failed", "failed to encode runtime events")
			return
		}
		rawEvents = append(rawEvents, raw)
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": rawEvents})
}

func (h *RuntimeEventsHandler) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "stream_unsupported", "streaming is not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	stream := h.hub.Subscribe(r.Context(), runtimeEventFilterFromRequest(r))
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case event, ok := <-stream:
			if !ok {
				return
			}
			raw, err := runtimeevents.MarshalJSON(event)
			if err != nil {
				continue
			}
			if event.Id != "" {
				_, _ = fmt.Fprintf(w, "id: %s\n", event.Id)
			}
			_, _ = fmt.Fprint(w, "event: runtime-event\n")
			_, _ = fmt.Fprintf(w, "data: %s\n\n", raw)
			flusher.Flush()
		}
	}
}

func runtimeEventFilterFromRequest(r *http.Request) runtimeevents.Filter {
	filter := runtimeevents.Filter{
		WorkflowID: r.URL.Query().Get("workflowId"),
		OpID:       r.URL.Query().Get("opId"),
		Site:       r.URL.Query().Get("site"),
		WorkerID:   r.URL.Query().Get("workerId"),
	}
	if limitText := r.URL.Query().Get("limit"); limitText != "" {
		if limit, err := strconv.Atoi(limitText); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}
	return filter
}

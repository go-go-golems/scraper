package runtimeevents

import (
	"context"
	"sync"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"google.golang.org/protobuf/proto"
)

type Filter struct {
	WorkflowID string
	OpID       string
	Site       string
	WorkerID   string
	Limit      int
}

type Hub struct {
	mu          sync.RWMutex
	capacity    int
	events      []*runtimev1.RuntimeEventV1
	subscribers map[chan *runtimev1.RuntimeEventV1]Filter
}

func NewHub(capacity int) *Hub {
	if capacity <= 0 {
		capacity = DefaultRecentEventLimit
	}
	return &Hub{
		capacity:    capacity,
		subscribers: map[chan *runtimev1.RuntimeEventV1]Filter{},
	}
}

func (h *Hub) Add(event *runtimev1.RuntimeEventV1) {
	if h == nil || event == nil {
		return
	}

	cloned := proto.Clone(event).(*runtimev1.RuntimeEventV1)

	h.mu.Lock()
	h.events = append(h.events, cloned)
	if len(h.events) > h.capacity {
		h.events = append([]*runtimev1.RuntimeEventV1(nil), h.events[len(h.events)-h.capacity:]...)
	}

	deliveries := make([]struct {
		ch    chan *runtimev1.RuntimeEventV1
		event *runtimev1.RuntimeEventV1
	}, 0, len(h.subscribers))
	for ch, filter := range h.subscribers {
		if !matchesFilter(cloned, filter) {
			continue
		}
		deliveries = append(deliveries, struct {
			ch    chan *runtimev1.RuntimeEventV1
			event *runtimev1.RuntimeEventV1
		}{
			ch:    ch,
			event: proto.Clone(cloned).(*runtimev1.RuntimeEventV1),
		})
	}
	h.mu.Unlock()

	for _, delivery := range deliveries {
		select {
		case delivery.ch <- delivery.event:
		default:
		}
	}
}

func (h *Hub) Recent(filter Filter) []*runtimev1.RuntimeEventV1 {
	if h == nil {
		return nil
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	matches := make([]*runtimev1.RuntimeEventV1, 0, len(h.events))
	for _, event := range h.events {
		if !matchesFilter(event, filter) {
			continue
		}
		matches = append(matches, proto.Clone(event).(*runtimev1.RuntimeEventV1))
	}

	if filter.Limit > 0 && len(matches) > filter.Limit {
		matches = matches[len(matches)-filter.Limit:]
	}
	return matches
}

func (h *Hub) Subscribe(ctx context.Context, filter Filter) <-chan *runtimev1.RuntimeEventV1 {
	if h == nil {
		closed := make(chan *runtimev1.RuntimeEventV1)
		close(closed)
		return closed
	}

	ch := make(chan *runtimev1.RuntimeEventV1, 32)
	h.mu.Lock()
	h.subscribers[ch] = filter
	h.mu.Unlock()

	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.subscribers, ch)
		close(ch)
		h.mu.Unlock()
	}()

	return ch
}

func matchesFilter(event *runtimev1.RuntimeEventV1, filter Filter) bool {
	if event == nil {
		return false
	}
	if filter.WorkflowID != "" && event.WorkflowId != filter.WorkflowID {
		return false
	}
	if filter.OpID != "" && event.OpId != filter.OpID {
		return false
	}
	if filter.Site != "" && event.Site != filter.Site {
		return false
	}
	if filter.WorkerID != "" && event.WorkerId != filter.WorkerID {
		return false
	}
	return true
}

package runtimeevents

import (
	"context"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
)

// Publisher is the producer-facing runtime-event sink used by scraper code.
// Implementations may distribute events through sessionstream, tests, or no-op
// adapters. Callers should pass their active request/worker context so the
// sessionstream command path can preserve cancellation and tracing semantics.
type Publisher interface {
	Publish(ctx context.Context, event *runtimev1.RuntimeEventV1) error
}

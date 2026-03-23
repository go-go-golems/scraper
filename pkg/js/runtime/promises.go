package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
)

func WaitForPromise(ctx context.Context, runtime *gggengine.Runtime, promise *goja.Promise) (any, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		ret, err := runtime.Owner.Call(ctx, "scraper.js.promise", func(_ context.Context, vm *goja.Runtime) (any, error) {
			return promiseSnapshot{
				State:  promise.State(),
				Result: promise.Result(),
			}, nil
		})
		if err != nil {
			return nil, err
		}

		snapshot := ret.(promiseSnapshot)
		switch snapshot.State {
		case goja.PromiseStatePending:
			time.Sleep(5 * time.Millisecond)
			continue
		case goja.PromiseStateFulfilled:
			if snapshot.Result == nil || goja.IsUndefined(snapshot.Result) || goja.IsNull(snapshot.Result) {
				return nil, nil
			}
			return snapshot.Result.Export(), nil
		case goja.PromiseStateRejected:
			if snapshot.Result == nil || goja.IsUndefined(snapshot.Result) || goja.IsNull(snapshot.Result) {
				return nil, fmt.Errorf("promise rejected")
			}
			return nil, fmt.Errorf("promise rejected: %v", snapshot.Result.Export())
		default:
			return nil, fmt.Errorf("unknown promise state %v", snapshot.State)
		}
	}
}

type promiseSnapshot struct {
	State  goja.PromiseState
	Result goja.Value
}

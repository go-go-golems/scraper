//go:build ignore

// Probe single-process scheduler concurrency in scraper v0.0.4.
// Run from the scraper module:
//
//	go run ./ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/02-probe-single-process-concurrency.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
)

type sleepingRunner struct{ active, maximum atomic.Int64 }

func (r *sleepingRunner) Kind() string { return "sleep" }
func (r *sleepingRunner) Run(ctx context.Context, rc runner.RunContext) (*model.OpResult, error) {
	active := r.active.Add(1)
	defer r.active.Add(-1)
	for {
		maximum := r.maximum.Load()
		if active <= maximum || r.maximum.CompareAndSwap(maximum, active) {
			break
		}
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}
	return &model.OpResult{OpID: rc.Op.ID}, nil
}

func main() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "scraper-concurrency-probe-*")
	must(err)
	defer os.RemoveAll(dir)
	store, err := sqlitestore.Open(ctx, filepath.Join(dir, "engine.db"))
	must(err)
	defer store.Close()
	impl := &sleepingRunner{}
	registry := runner.NewRegistry()
	must(registry.Register(impl))
	sched, err := scheduler.New(store, registry, scheduler.Config{MaxWorkers: 3, PollInterval: time.Millisecond, DefaultLeaseDuration: time.Minute}, "probe-worker", nil)
	must(err)
	sched.SetQueuePolicyProvider(func(context.Context, model.SiteName, model.QueueKey) model.QueuePolicy {
		return model.QueuePolicy{MaxInFlight: 3}
	})
	ops := []model.OpSpec{}
	for i := 0; i < 3; i++ {
		ops = append(ops, model.OpSpec{ID: model.OpID(fmt.Sprintf("batch-%d", i)), WorkflowID: "concurrency-probe", Site: "probe", Kind: "sleep", Queue: "provider"})
	}
	must(sched.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{Workflow: model.WorkflowRun{ID: "concurrency-probe", Site: "probe", Name: "concurrency probe"}, Initial: ops}))
	started := time.Now()
	cycle, err := sched.RunOnce(ctx)
	must(err)
	fmt.Printf("processed=%d max_active=%d elapsed_ms=%d\n", cycle.Processed, impl.maximum.Load(), time.Since(started).Milliseconds())
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

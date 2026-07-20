//go:build ignore

// Probe failure isolation and manual-retry descendant behavior in scraper v0.0.4.
// Run from the scraper module:
//
//	go run ./ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/01-probe-retry-descendants.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
	"github.com/go-go-golems/scraper/pkg/engine/scheduler"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/go-go-golems/scraper/pkg/services/engineview"
)

type probeRunner struct{ attempts map[model.OpID]int }

func (r *probeRunner) Kind() string { return "probe" }
func (r *probeRunner) Run(_ context.Context, rc runner.RunContext) (*model.OpResult, error) {
	r.attempts[rc.Op.ID]++
	if rc.Op.ID == "batch-a" && r.attempts[rc.Op.ID] == 1 {
		return &model.OpResult{OpID: rc.Op.ID, Error: &model.OpError{
			Code: "malformed_provider_output", Message: "injected first-attempt failure", Retryable: false,
		}}, nil
	}
	body, _ := json.Marshal(map[string]any{"attempt": r.attempts[rc.Op.ID], "op": rc.Op.ID})
	return &model.OpResult{OpID: rc.Op.ID, Data: body}, nil
}

func main() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "scraper-retry-probe-*")
	must(err)
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "engine.db")
	store, err := sqlitestore.Open(ctx, dbPath)
	must(err)
	defer store.Close()

	registry := runner.NewRegistry()
	must(registry.Register(&probeRunner{attempts: map[model.OpID]int{}}))
	sched, err := scheduler.New(store, registry, scheduler.Config{
		MaxWorkers: 3, PollInterval: time.Millisecond, DefaultLeaseDuration: time.Minute,
	}, "probe-worker", nil)
	must(err)
	sched.SetQueuePolicyProvider(func(context.Context, model.SiteName, model.QueueKey) model.QueuePolicy {
		return model.QueuePolicy{MaxInFlight: 3}
	})

	workflowID := model.WorkflowID("retry-descendant-probe")
	batchA, batchB := model.OpID("batch-a"), model.OpID("batch-b")
	must(sched.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{ID: workflowID, Site: "probe", Name: "retry descendant probe"},
		Initial: []model.OpSpec{
			{ID: batchA, WorkflowID: workflowID, Site: "probe", Kind: "probe", Queue: "provider"},
			{ID: batchB, WorkflowID: workflowID, Site: "probe", Kind: "probe", Queue: "provider"},
			{ID: "finalize", WorkflowID: workflowID, Site: "probe", Kind: "probe", Queue: "finalize", DependsOn: []model.Dependency{{OpID: batchA, Required: true}, {OpID: batchB, Required: true}}},
		},
	}))

	first, err := sched.RunOnce(ctx)
	must(err)
	fmt.Printf("first cycle: processed=%d succeeded=%d failed=%d\n", first.Processed, first.Succeeded, first.Failed)
	printState(ctx, dbPath, workflowID)

	must(engineview.NewService(dbPath).RetryOp(ctx, workflowID, batchA))
	second, err := sched.RunOnce(ctx)
	must(err)
	fmt.Printf("after manual retry: processed=%d succeeded=%d failed=%d\n", second.Processed, second.Succeeded, second.Failed)
	printState(ctx, dbPath, workflowID)
}

func printState(ctx context.Context, dbPath string, workflowID model.WorkflowID) {
	view := engineview.NewService(dbPath)
	summary, err := view.Workflow(ctx, workflowID)
	must(err)
	ops, err := view.WorkflowOps(ctx, workflowID)
	must(err)
	fmt.Printf("workflow=%s", summary.Workflow.Status)
	for _, op := range ops {
		fmt.Printf(" %s=%s", op.Op.ID, op.Status)
	}
	fmt.Println()
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

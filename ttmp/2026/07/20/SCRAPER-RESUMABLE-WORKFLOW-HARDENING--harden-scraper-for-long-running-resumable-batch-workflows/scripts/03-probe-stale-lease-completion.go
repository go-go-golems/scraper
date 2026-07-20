//go:build ignore

// Probe repeated heartbeat and stale-worker completion rejection after the
// SCRAPER-RESUMABLE-WORKFLOW-HARDENING lease safety implementation.
// Run from the scraper module:
//
//	go run ./ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/03-probe-stale-lease-completion.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/go-go-golems/scraper/pkg/services/engineview"
)

func main() {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "scraper-stale-lease-probe-*")
	must(err)
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "engine.db")
	store, err := sqlitestore.Open(ctx, dbPath)
	must(err)
	defer store.Close()

	workflowID := model.WorkflowID("stale-lease-probe")
	opID := model.OpID("batch-0")
	must(store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{ID: workflowID, Site: "probe", Name: "stale lease probe", Status: model.WorkflowStatusPending},
		Initial:  []model.OpSpec{{ID: opID, WorkflowID: workflowID, Site: "probe", Kind: "probe", Queue: "provider"}},
	}))

	// The SQLite store now compares epoch-microsecond columns, so this mixed
	// precision value is intentionally safe.
	t0 := time.Date(2026, 7, 20, 18, 0, 0, 123456789, time.UTC)
	_, err = store.RefreshRunnableOps(ctx, t0)
	must(err)
	_, lease1, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID: "worker-1", Queue: "provider", Site: "probe",
		Policy: model.QueuePolicy{MaxInFlight: 1}, LeaseDuration: time.Second, Now: t0,
	})
	must(err)
	if lease1 == nil {
		panic("first lease not acquired")
	}

	lease1, err = store.HeartbeatLease(ctx, opID, *lease1, t0.Add(500*time.Millisecond), time.Second)
	must(err)
	lease1, err = store.HeartbeatLease(ctx, opID, *lease1, t0.Add(time.Second), time.Second)
	must(err)
	printLease(ctx, dbPath, workflowID, "after two cumulative heartbeats")

	// The current lease expires at t0+2s, so a later worker may safely acquire it.
	_, err = store.RefreshRunnableOps(ctx, t0.Add(2500*time.Millisecond))
	must(err)
	_, lease2, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID: "worker-2", Queue: "provider", Site: "probe",
		Policy: model.QueuePolicy{MaxInFlight: 1}, LeaseDuration: time.Second, Now: t0.Add(2500 * time.Millisecond),
	})
	must(err)
	if lease2 == nil {
		panic("second lease not acquired")
	}
	fmt.Printf("re-leased: old_token=%s new_token=%s\n", lease1.Token, lease2.Token)

	oldData, _ := json.Marshal(map[string]string{"writer": "stale-worker-1"})
	err = store.CompleteOp(ctx, opID, storecontract.Completion{Lease: *lease1, Result: model.OpResult{OpID: opID, Data: oldData, CompletedAt: t0.Add(2600 * time.Millisecond)}})
	fmt.Printf("stale completion rejected=%v\n", err)
	printLease(ctx, dbPath, workflowID, "after stale completion")

	newData, _ := json.Marshal(map[string]string{"writer": "current-worker-2"})
	err = store.CompleteOp(ctx, opID, storecontract.Completion{Lease: *lease2, Result: model.OpResult{OpID: opID, Data: newData, CompletedAt: t0.Add(2700 * time.Millisecond)}})
	fmt.Printf("current completion error=%v\n", err)
	result, err := store.GetResult(ctx, workflowID, opID)
	must(err)
	fmt.Printf("final result=%s\n", result.Data)
}

func printLease(ctx context.Context, dbPath string, workflowID model.WorkflowID, label string) {
	ops, err := engineview.NewService(dbPath).WorkflowOps(ctx, workflowID)
	must(err)
	if len(ops) != 1 {
		panic(fmt.Sprintf("want one op, got %d", len(ops)))
	}
	lease := "none"
	if ops[0].Lease != nil {
		lease = fmt.Sprintf("worker=%s expires=%s", ops[0].Lease.WorkerID, ops[0].Lease.ExpiresAt.Format(time.RFC3339Nano))
	}
	fmt.Printf("%s: status=%s lease=%s\n", label, ops[0].Status, lease)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

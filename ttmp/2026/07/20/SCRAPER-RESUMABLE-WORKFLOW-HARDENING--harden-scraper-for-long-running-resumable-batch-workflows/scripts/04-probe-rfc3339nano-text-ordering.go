//go:build ignore

// Probe that mixed RFC3339Nano precision no longer affects scheduling because
// scraper uses sortable epoch-microsecond columns for SQLite comparisons.
// Run from the scraper module:
//
//	go run ./ttmp/2026/07/20/SCRAPER-RESUMABLE-WORKFLOW-HARDENING--harden-scraper-for-long-running-resumable-batch-workflows/scripts/04-probe-rfc3339nano-text-ordering.go
package main

import (
	"context"
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
	dir, err := os.MkdirTemp("", "scraper-time-order-probe-*")
	must(err)
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "engine.db")
	store, err := sqlitestore.Open(ctx, dbPath)
	must(err)
	defer store.Close()

	workflowID := model.WorkflowID("time-order-probe")
	opID := model.OpID("op-0")
	must(store.CreateWorkflow(ctx, storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{ID: workflowID, Site: "probe", Name: "time order probe"},
		Initial:  []model.OpSpec{{ID: opID, WorkflowID: workflowID, Site: "probe", Kind: "probe", Queue: "q"}},
	}))

	// The formatted values retain the historically unsafe lexical ordering, but
	// refresh is expected to recover the lease using integer epoch time.
	t0 := time.Date(2026, 7, 20, 18, 0, 0, 0, time.UTC)
	_, err = store.RefreshRunnableOps(ctx, t0)
	must(err)
	_, lease, err := store.LeaseReadyOp(ctx, storecontract.LeaseRequest{
		WorkerID: "worker", Queue: "q", Site: "probe",
		Policy: model.QueuePolicy{MaxInFlight: 1}, LeaseDuration: time.Second, Now: t0,
	})
	must(err)
	fmt.Printf("expires=%s refresh_at=%s chronological_expired=%t lexical_expired=%t\n",
		lease.ExpiresAt.Format(time.RFC3339Nano),
		t0.Add(1500*time.Millisecond).Format(time.RFC3339Nano),
		lease.ExpiresAt.Before(t0.Add(1500*time.Millisecond)),
		lease.ExpiresAt.Format(time.RFC3339Nano) <= t0.Add(1500*time.Millisecond).Format(time.RFC3339Nano),
	)

	changed, err := store.RefreshRunnableOps(ctx, t0.Add(1500*time.Millisecond))
	must(err)
	ops, err := engineview.NewService(dbPath).WorkflowOps(ctx, workflowID)
	must(err)
	fmt.Printf("integer-time refresh_changed=%d status=%s lease_present=%t\n", changed, ops[0].Status, ops[0].Lease != nil)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

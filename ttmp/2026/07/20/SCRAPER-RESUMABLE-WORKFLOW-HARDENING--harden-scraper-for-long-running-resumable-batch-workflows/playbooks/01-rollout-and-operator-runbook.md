---
Title: Resumable workflow hardening rollout and operator runbook
Ticket: SCRAPER-RESUMABLE-WORKFLOW-HARDENING
Status: active
Topics:
    - scraper
    - scheduler
    - worker
    - sqlite
    - workflows
DocType: playbook
Intent: long-term
Owners: []
RelatedFiles:
    - Path: repo://pkg/engine/scheduler/scheduler.go
      Note: Lease heartbeat and concurrency behavior
    - Path: repo://pkg/engine/store/sqlite/migrations/003_sortable_timestamp_columns.sql
      Note: Upgrade migration
    - Path: repo://pkg/services/engineview/workflow_mutation_service.go
      Note: Retry and cancellation procedures
    - Path: repo://pkg/workflow/runtime.go
      Note: EnsureRun and snapshot operator APIs
ExternalSources: []
Summary: Release, upgrade, recovery, and validation procedures for hardened scraper workflow execution.
LastUpdated: 2026-07-20T21:12:00Z
WhatFor: Safely upgrade durable engine databases and operate long-running resumable workflows.
WhenToUse: Use before releasing or operating scraper workers backed by SQLite workflow state.
---


# Resumable workflow hardening rollout and operator runbook

## Compatibility

The upgrade adds schema migration 003. It backfills sortable epoch-microsecond columns from existing RFC3339Nano text values and leaves legacy text fields intact for compatibility. Worker scheduling, leases, retries, queue admission, and inspection use integer timestamp columns after migration.

Operation state now includes `blocked`:

- `blocked` is dependency-derived and may reopen after explicit repair of every required dependency.
- `canceled` is an explicit terminal operator action and never reopens through dependency refresh.
- Existing workflows with a failed required dependency will become `blocked` on their next scheduler refresh rather than receiving new dependency-derived `canceled` transitions.

`MaxWorkers` now means actual concurrent in-process runner capacity. SQLite writes remain short serialized transactions; provider/executor work is concurrent. An emitted child operation is admitted in the following scheduler cycle, not necessarily the parent’s current leasing snapshot.

## Pre-release validation

```bash
cd /home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/scraper
GOWORK=off go test ./... -count=1
GOWORK=off go test -race ./pkg/engine/scheduler ./pkg/engine/store/sqlite ./pkg/workflow ./pkg/services/engineview -count=1
GOWORK=off go vet ./...
GOWORK=off gosec ./pkg/engine/store/sqlite
```

The workspace’s local Goja checkout is presently incompatible with scraper’s pinned `goja_nodejs`; use `GOWORK=off` until that unrelated workspace dependency alignment is intentionally released.

## Upgrade procedure

1. Stop or drain every worker using the target engine database. Do not run old and new binaries against the same database during the migration rollout.
2. Take a filesystem-consistent SQLite backup including `engine.db`, `engine.db-wal`, and `engine.db-shm` if present.
3. Start one new binary or run an inspection command that opens the database. Migration 003 executes transactionally and parses every legacy timestamp.
4. Verify migration and state:

   ```bash
   scraper engine status --engine-db state/engine.db
   scraper workflow status --engine-db state/engine.db --workflow-id WORKFLOW_ID
   ```

5. Start one worker with deliberately small `MaxWorkers` and queue caps. Confirm snapshots, leases, and event delivery before increasing throughput.

If migration parsing fails, stop. Preserve the database and report the exact malformed timestamp. Do not manually mark migration 003 applied.

## Starting or attaching durable work

Use `EnsureRun` for immutable batch identities. Include version/schema, input digest, model/pipeline configuration digest, and recovery policy in the identity; exclude secrets and raw sensitive content.

```go
handle, err := runtime.EnsureRun(ctx, "batch-package", input,
    workflow.WithRunIdentity(map[string]string{
        "inputDigest": inputDigest,
        "pipeline": "v2",
    }),
)
```

`handle.Created` distinguishes new work from attachment. The same canonical identity returns the same workflow. Reusing the ID with a different identity returns an identity conflict and must be investigated rather than retried with a random new ID.

## Recovering a failed batch

1. Inspect the workflow snapshot and operation errors.
2. Correct the underlying input/configuration only when it remains compatible with the immutable workflow identity.
3. Invoke `RetryStep`/the operator retry endpoint for the failed operation.
4. Verify required descendants transition `blocked -> pending -> ready` only after all required dependencies succeed.
5. Do not use retry to reopen a workflow intentionally canceled by an operator; start a new identity when business policy requires new work.

## Lease-loss response

The scheduler heartbeats every configured interval (default one third of lease duration). A runner that loses its lease receives a canceled context and cannot commit a completion/failure transition. The event is published as a failed runtime event with error code `lease_lost`.

On lease loss:

1. Inspect queue/database availability and worker clock behavior.
2. Let the expired lease refresh to `ready`, or investigate the current lease owner.
3. Ensure executors tolerate at-least-once external effects using stable idempotency/output identities.
4. Never manually write results for the stale worker token.

## Dashboard and observers

Observers are post-commit notifications, are serialized, and observer panics cannot abort scheduling. They are not durable history. Use `Runtime.Snapshot` and `Runtime.SnapshotsSince` to reconstruct truth after restart; sessionstream/runtime-event transports are delivery layers only.

## Rollback

Binary rollback after migration is not supported unless the older binary understands schema migration 003’s additive columns. Restore the pre-upgrade SQLite backup if rollback is necessary. Never delete only the new integer columns from a live database.

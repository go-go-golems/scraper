# Tasks

## Analysis And Preparation

- [x] Create the ticket workspace.
- [x] Map the current responsibilities inside `pkg/engine/store/sqlite/store.go`.
- [x] Map the current responsibilities inside `pkg/services/engineview/service.go`.
- [x] Write the decomposition design and target file layout.
- [x] Record the investigation diary.

## Store Decomposition

- [x] Extract workflow-specific methods into `workflow_store.go`.
- [x] Extract op enqueue/read helpers into `op_store.go`.
- [ ] Extract leasing and heartbeat logic into `lease_store.go`.
- [x] Extract result/artifact logic into `result_store.go`.
- [x] Extract queue limiter logic into `queue_limiter.go`.
- [x] Extract JSON/sql helper functions into `sql_helpers.go`.
- [ ] Keep the public `Store` type stable during the move-only pass.

## Engine View Decomposition

- [x] Extract workflow reads into `workflow_read_service.go`.
- [x] Extract queue reads into `queue_read_service.go`.
- [x] Extract artifact and op-result reads into `artifact_read_service.go`.
- [x] Extract retry/cancel helpers into `workflow_mutation_service.go`.
- [x] Extract DB-opening and existence helpers into `db_helpers.go`.

## Validation

- [x] Keep `go test ./pkg/engine/store/sqlite -count=1` green during the split.
- [ ] Keep `go test ./pkg/services/engineview ./pkg/api/server -count=1` green during the split.
- [ ] Run `go test ./... -count=1` at the end.
- [ ] Run `docmgr doctor --ticket SCRAPER-CLEANUP-STORE-VIEW --stale-after 30`.

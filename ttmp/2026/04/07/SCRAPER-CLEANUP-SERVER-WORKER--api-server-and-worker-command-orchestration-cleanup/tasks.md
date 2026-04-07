# Tasks

## Analysis And Design

- [x] Create the ticket workspace.
- [x] Review the concerns currently mixed into `pkg/api/server/server.go`.
- [x] Review the concerns currently mixed into `pkg/cmd/worker.go`.
- [x] Write the orchestration cleanup design and target file layout.
- [x] Record the investigation diary.

## API Server Cleanup

- [x] Split route registration by domain into dedicated files.
- [ ] Split request logging and metrics middleware into its own file.
- [ ] Split runtime-event router startup into its own file.
- [ ] Keep `server.New(...)` as the composition root.

## Worker Command Cleanup

- [ ] Keep Cobra command and flags in `worker.go`.
- [ ] Move worker runtime setup into `worker_runtime.go`.
- [ ] Move metrics listener boot into `worker_metrics.go`.
- [ ] Move observer composition into `worker_observers.go`.

## Validation

- [x] Run `go test ./pkg/api/server ./pkg/cmd -count=1`.
- [ ] Run `go test ./... -count=1`.
- [ ] Run `docmgr doctor --ticket SCRAPER-CLEANUP-SERVER-WORKER --stale-after 30`.

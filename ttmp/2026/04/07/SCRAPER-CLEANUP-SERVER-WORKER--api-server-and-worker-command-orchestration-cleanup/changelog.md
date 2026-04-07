# Changelog

## 2026-04-07

- Initial workspace created
- Added the orchestration cleanup design and task plan for the API server and worker command files.
- Split API server route registration by domain into dedicated route files while keeping `server.New(...)` as the composition root.
- Split request logging, request metrics, and the HTTP status recorder into `middleware_request.go`.
- Split runtime-event router startup into `runtime_event_router.go`, leaving `server.New(...)` as the clear API composition root.
- Split worker observer composition into `worker_observers.go`.
- Split worker metrics listener boot into `worker_metrics.go`.
- Split the worker runtime setup into `worker_runtime.go`, leaving `worker.go` focused on Cobra command construction and flags.

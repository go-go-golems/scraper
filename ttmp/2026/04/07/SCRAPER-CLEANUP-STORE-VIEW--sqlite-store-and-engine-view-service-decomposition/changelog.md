# Changelog

## 2026-04-07

- Initial workspace created
- Added the decomposition design and task plan for the SQLite store and engine view service.
- Extracted artifact/op-result reads and shared DB helpers out of `pkg/services/engineview/service.go` into dedicated files without changing behavior.
- Extracted workflow reads, queue reads, and retry/cancel helpers out of `pkg/services/engineview/service.go`, leaving it as a thin facade plus `EngineStatus`.
- Extracted SQLite queue limiter logic and shared SQL/JSON helpers into dedicated store files without changing behavior.
- Extracted SQLite result completion, failure handling, result loading, and artifact loading into `result_store.go`.
- Extracted workflow creation, reads, status updates, and workflow stats into `workflow_store.go`.

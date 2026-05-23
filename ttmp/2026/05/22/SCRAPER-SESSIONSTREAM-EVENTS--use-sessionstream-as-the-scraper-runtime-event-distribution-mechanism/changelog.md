# Changelog

## 2026-05-22

- Initial workspace created


## 2026-05-22

Created evidence-backed intern design guide and investigation diary for migrating scraper runtime event distribution to sessionstream.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/05/22/SCRAPER-SESSIONSTREAM-EVENTS--use-sessionstream-as-the-scraper-runtime-event-distribution-mechanism/design-doc/01-intern-guide-to-sessionstream-backed-scraper-runtime-events.md — Primary design and implementation guide
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/05/22/SCRAPER-SESSIONSTREAM-EVENTS--use-sessionstream-as-the-scraper-runtime-event-distribution-mechanism/reference/01-investigation-diary.md — Chronological diary for this investigation


## 2026-05-22

Validated ticket documentation with docmgr doctor; all checks passed.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/05/22/SCRAPER-SESSIONSTREAM-EVENTS--use-sessionstream-as-the-scraper-runtime-event-distribution-mechanism/index.md — Ticket index validated by docmgr doctor


## 2026-05-22

Prepared final ticket bundle for reMarkable upload after validation and diary updates.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/05/22/SCRAPER-SESSIONSTREAM-EVENTS--use-sessionstream-as-the-scraper-runtime-event-distribution-mechanism/reference/01-investigation-diary.md — Delivery evidence and upload notes
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/05/22/SCRAPER-SESSIONSTREAM-EVENTS--use-sessionstream-as-the-scraper-runtime-event-distribution-mechanism/tasks.md — Task completion state for final handoff


## 2026-05-22

Revised the design to remove backwards-compatibility requirements and to follow Pinocchio's protobuf-defined sessionstream app pattern.

### Related Files

- /home/manuel/code/wesen/go-go-golems/pinocchio/pkg/chatapp/chat.go — Reference implementation for schema registration and installation
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/05/22/SCRAPER-SESSIONSTREAM-EVENTS--use-sessionstream-as-the-scraper-runtime-event-distribution-mechanism/design-doc/01-intern-guide-to-sessionstream-backed-scraper-runtime-events.md — Updated design constraints


## 2026-05-22

Re-uploaded the revised no-compatibility Pinocchio-informed bundle to reMarkable.

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/ttmp/2026/05/22/SCRAPER-SESSIONSTREAM-EVENTS--use-sessionstream-as-the-scraper-runtime-event-distribution-mechanism/reference/01-investigation-diary.md — Updated delivery evidence for revised bundle


## 2026-05-22

Phase 1: added scraper sessionstream protobuf contracts, generated bindings, adapter package, and tests (commit 0ea7c29071279544366f5878edf34ac79c63d0db).

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/runtimeevents/sessionstream/projections.go — Runtime event UI/timeline projections
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/runtimeevents/sessionstream/publisher.go — Context-aware runtime event publisher and command handler
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/runtimeevents/sessionstream/runtime.go — Producer/server sessionstream runtime wiring
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/runtimeevents/sessionstream/runtime_test.go — Local and gochannel integration coverage
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/proto/scraper/runtime/sessionstream/v1/runtime_stream.proto — New scraper sessionstream protobuf contracts


## 2026-05-22

Phase 2: replaced backend REST/SSE runtime-event infrastructure with sessionstream websocket runtime and context-aware producers (commit ee5f4ba936ee0f5ce49d7d9f7d988855518ae567).

### Related Files

- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/server/routes_runtime_events.go — Runtime events route now registers only websocket endpoint
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/server/server.go — API now creates sessionstream runtime and websocket route
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/api/server/server_test.go — Websocket snapshot/live runtime event integration test
- /home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/runtimeevents/publisher.go — Context-aware runtime event publisher interface


---
Title: Workflow V3 Minimal Runtime
Slug: scraper-workflow-v3-minimal-runtime
Short: "Explains the first executable workflow-v3 slice, its JavaScript DSL, exact task bundles, and durable privacy boundaries."
Topics:
- scraper
- runtime
- workflows
- javascript
- workers
Commands:
- worker
- engine
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

Workflow v3 now has a minimal executable vertical slice for trusted first-party
JavaScript tasks. It is intentionally separate from the existing v2 site and
submission runtime while its contracts are exercised and expanded.

## What runs today

A CommonJS authoring script can import `workflow` and a descriptor-only task
module, then use:

- `workflow.define(name, callback)`;
- `plan.input(name, {schema})`;
- `plan.task(nodeKey, taskDescriptor, callback?)`;
- `job.after(otherJob)`;
- `job.output(port)`;
- `plan.output(name, value)`;
- `workflow.validate`, `workflow.digest`, `workflow.toIR`, and
  `workflow.compile`.

The JavaScript builder produces the same canonical Go IR as direct Go
construction. Compilation pins task kind, task version, bundle digest,
entrypoint, task ABI, schemas, declared modules, catalog digest, IR digest, and
plan digest.

## Execution boundary

The first task profile supports the current go-go-goja `fs` module as the alias
`fs:input`. Each attempt receives:

- a fresh Goja runtime and CommonJS module cache;
- only `workflow/task` and modules declared by the pinned task;
- a read-only temporary filesystem containing only bound input artifacts;
- a context with immutable input refs, identity, cancellation checkpoint, and
  validating output writers.

Source bytes live in an external content-addressed artifact store. SQLite stores
only schemas, digests, media types, sizes, bounded locators, plan identities,
lease state, attempt outcomes, and redacted events.

## Durable behavior

The v3 SQLite store uses `(run_id, node_key)` identities and append-only attempt
numbers. Lease expiry creates a `lease_lost` outcome and a later attempt.
Completion requires the current lease token and cancellation epoch. A stale,
expired, or canceled attempt cannot publish output refs.

Workers match the complete implementation identity before leasing. A worker
with the same task kind/version but different bundle bytes, entrypoint, or ABI
cannot claim the node.

## Validation

Run the focused implementation suites with:

```text
GOWORK=off go test ./pkg/workflowv3 \
  ./pkg/gojamodules/workflow \
  ./pkg/workflowv3runtime \
  ./pkg/workflowv3sqlite -count=1

GOWORK=off go test -race \
  ./pkg/workflowv3runtime \
  ./pkg/workflowv3sqlite -count=1
```

The end-to-end test authors a two-node workflow through JavaScript, processes a
12,000-row JSONL artifact, restarts between nodes, reopens the final output, and
scans SQLite main/WAL/SHM bytes for source and secret canaries.

## Source map

- `pkg/workflowv3` — canonical contracts, compiler, artifacts, bundles, and
  registry.
- `pkg/gojamodules/workflow` — safe authoring module and TypeScript declaration.
- `pkg/workflowv3runtime` — fresh-runtime task runner and sequential vertical
  slice engine.
- `pkg/workflowv3sqlite` — compact durable store and fencing.
- `pkg/testfixtures/workflowv3linear` — paired workflow and task bundle used by
  authoring and execution tests.

Later slices add HTTP, work-conserving resource dispatch, database side effects,
lazy maps, bounded reductions, budgets, gates, and stronger process isolation.
The minimal slice does not translate or silently accept v2 raw operations.

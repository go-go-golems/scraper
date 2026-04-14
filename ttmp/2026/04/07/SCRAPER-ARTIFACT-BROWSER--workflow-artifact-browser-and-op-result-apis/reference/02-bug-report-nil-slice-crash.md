---
Title: "Bug: OpResult nil slice crash — EmittedIDs.length TypeError"
Ticket: SCRAPER-ARTIFACT-BROWSER
Status: done
Topics:
    - bug
    - frontend
    - backend
    - http-api
DocType: reference
Intent: long-term
Owners: []
RelatedFiles:
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/store/sqlite/result_store.go"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/model/types.go"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/components/workflows/op-detail/OpResultTab.tsx"
    - "/home/manuel/workspaces/2026-03-23/js-scraper/scraper/web/src/api/types.ts"
Summary: "OpResultTab crashes with TypeError: can't access property length, result.EmittedIDs is null when opening the Result tab for ops whose result has nil slice fields."
LastUpdated: 2026-04-07T20:15:00-04:00
WhatFor: "Track this bug, its root cause, and the fix so it doesn't regress."
WhenToUse: "Reference when reviewing similar nil/empty slice serialization issues, or when auditing OpResult field handling."
---

# Bug: `OpResult` nil slice crash

## Summary

`OpResultTab` crashes with:

```
TypeError: can't access property "length", result.EmittedIDs is null
    OpResultTab OpResultTab.tsx:63
```

The crash occurs on the "Result" tab of `OpDetailDrawer` for any op whose stored result has nil slice fields.

## Steps to reproduce

1. Run a scraper workflow that produces an op result (e.g., `frontpage-extract`).
2. Open the workflow detail page.
3. Click an op to open `OpDetailDrawer`.
4. Switch to the "Result" tab.
5. Observe the crash and white screen.

Note: the crash only manifests for ops whose result was written with nil slice fields (e.g., `EmittedIDs`, `Records`, `Artifacts`, `Emitted`). Ops whose result was written with populated slices do not crash.

## Root cause

Go's `encoding/json` serializes **nil slices as `null`**, not `[]`.

In `model.OpResult`, the following fields are typed as slices but were left nil at construction time:

- `Records     []RecordWrite`
- `Artifacts   []ArtifactWrite`
- `Emitted     []OpSpec`
- `EmittedIDs  []OpID`

This happens in two places:

1. **Scheduler fallback** (`scheduler.go`): when the runner returns a nil result, the scheduler creates a minimal `&model.OpResult{OpID: op.ID, CompletedAt: now}` — all slices are nil.
2. **DB retrieval** (`result_store.go`): when the DB columns for those fields are NULL, `unmarshalJSON` leaves the Go slice as nil.

When serialized to JSON over the HTTP API, these nil slices arrive in the browser as `null`. The TypeScript type `OpResult` declares them as `string[]` (non-nullable), so `.length` throws.

## Fix

### 1. `pkg/engine/store/sqlite/result_store.go`

After loading from DB, normalize nil slices to empty slices so they always serialize as `[]` not `null`:

```go
if result.Records == nil {
    result.Records = []model.RecordWrite{}
}
if result.Emitted == nil {
    result.Emitted = []model.OpSpec{}
}
if result.EmittedIDs == nil {
    result.EmittedIDs = []model.OpID{}
}
```

This is the authoritative persistence path and the one that serves the HTTP API.

### 2. `web/src/components/workflows/op-detail/OpResultTab.tsx`

Optional chaining as a defensive TS safety net — the type contract cannot be fully trusted across the API boundary:

```ts
result.EmittedIDs?.length ?? 0
result.Artifacts?.length ?? 0
```

### Why not `omitempty` on struct tags

Omitting nil slices via `json:",omitempty"` produces `undefined` in JSON, which also crashes on `.length`. The correct serialization target is `[]`, which requires initializing to empty slices, not omitting the field.

## Files changed

| File | Change |
|---|---|
| `pkg/engine/store/sqlite/result_store.go` | Nil-guard for `Records`, `Emitted`, `EmittedIDs` after DB load |
| `pkg/engine/model/types.go` | No change needed — struct tags are `json:"field"` (not `omitempty`) |
| `web/src/components/workflows/op-detail/OpResultTab.tsx` | `?.length ?? 0` safety net for `EmittedIDs` and `Artifacts` |

## Validation

```bash
go test ./... -count=1   # all pass
npx tsc --noEmit         # no type errors
```

## Related

- The `OpResult` type in `web/src/api/types.ts` also types `EmittedIDs`, `Records`, `Artifacts`, `Emitted` as non-nullable arrays. Consider adding `?` to make the types match the runtime reality, though the nil-guard in Go is the primary fix.
- A future pass should audit all `OpResult` field accesses in the frontend for similar null-crash risks (e.g., iterating over `result.Records`).

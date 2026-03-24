---
Title: JavaScript API Reference
Slug: scraper-js-api-reference
Short: "Complete reference for the ctx object, database modules, op specs, and return envelopes in both submit verbs and execution scripts."
Topics:
- scraper
- javascript
- api
- reference
Commands:
- site
- worker
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

The scraper exposes two JavaScript environments with different `ctx` APIs. Submit verbs run at submission time to seed durable work. Execution scripts run later through the worker. Both share `ctx.emit()` and `require("site-db")` but differ in what else is available.

This page is the complete API surface. Use it as a lookup reference when writing site scripts.

## Submission-Time Context (verbs/)

Submit verb functions receive a `ctx` object with:

### Properties

| Property | Type | Description |
|----------|------|-------------|
| `ctx.site` | string | Current site name |
| `ctx.now` | string | Current UTC timestamp in RFC3339Nano format |
| `ctx.values` | object | Parsed CLI flag values as `{flagName: value}` |
| `ctx.sections` | object | Flag values grouped by Glazed section as `{sectionSlug: {flagName: value}}` |
| `ctx.command` | object | `{name, fullPath, function, module, sourceFile}` |
| `ctx.workflow` | object | `{id, site, name, status, input, metadata}` |

### Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `ctx.log(...args)` | void | Log info-level message to stderr with site and workflow context |
| `ctx.emit(spec)` | string | Emit a durable op and return its ID. See OpSpec below |
| `ctx.setWorkflowName(name)` | void | Set the workflow display name |
| `ctx.setWorkflowMetadata(obj)` | void | Set or replace workflow metadata (string map) |
| `ctx.setTargetOpID(opID)` | void | Mark an op as the user-visible completion target |

### Return Envelope

A submit verb may return `undefined` or an object:

```javascript
return {
  data: { ... },              // arbitrary metadata about the submission
  targetOpID: "op-id",        // alternative to ctx.setTargetOpID()
  workflowName: "name",       // alternative to ctx.setWorkflowName()
  workflowMetadata: { ... }   // alternative to ctx.setWorkflowMetadata()
};
```

Values set via methods take precedence over the return envelope.

### Verb Metadata

Submit verbs are discovered from `__verb__` annotations:

```javascript
doc(`Long-form help text shown in the CLI.`);

__verb__("functionName", {
  command: "cli-name",        // optional: override the CLI subcommand name
  short: "One-line help",
  fields: {
    "flag-name": {
      type: "string",         // string, int, bool, float, stringList
      help: "Flag description",
      default: "value"
    }
  }
});

function functionName(ctx) {
  // ctx.values["flag-name"] contains the parsed value
}
```

## Execution-Time Context (scripts/)

Execution scripts are loaded as CommonJS modules. The module export (or `default` export) is called with a `ctx` object:

```javascript
module.exports = function(ctx) { ... };
// or
module.exports.default = function(ctx) { ... };
// or (async)
module.exports = async function(ctx) { ... };
```

### Properties

| Property | Type | Description |
|----------|------|-------------|
| `ctx.site` | string | Current site name |
| `ctx.now` | string | Current UTC timestamp in RFC3339Nano format |
| `ctx.input` | any | Decoded JSON input for this op |
| `ctx.workflow` | object | `{id, site, name, status, input, metadata}` |
| `ctx.op` | object | `{id, workflowID, site, kind, queue, dedupKey, metadata}` |
| `ctx.lease` | object | `{workerID, token, acquiredAt, expiresAt}` |

### Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `ctx.log(...args)` | void | Log info-level message with op context |
| `ctx.dep(opID)` | object or null | Retrieve a dependency result. See DependencyResult below |
| `ctx.emit(spec)` | string | Emit a child op and return its ID. See OpSpec below |
| `ctx.writeRecord(collection, key, data)` | void | Write a record to the op result |
| `ctx.writeArtifact(spec)` | string | Write an artifact and return its ID. See ArtifactSpec below |

### Async Support

Scripts can be `async` functions or return Promises. The runtime awaits the result before persisting:

```javascript
module.exports = async function(ctx) {
  const db = require("site-db");
  db.exec("INSERT INTO items (name) VALUES (?)", ctx.input.name);
  return { data: { inserted: true } };
};
```

### Return Envelope

Scripts may return `undefined` or an object:

```javascript
return {
  data: { ... },    // result data persisted as JSON
  error: {           // signal a failure
    code: "PARSE_ERROR",
    message: "...",
    retryable: true,
    details: { ... }
  }
};
```

If the return value is not an `{data, error}` envelope, it is treated as the data directly.

## OpSpec (ctx.emit)

Both submit verbs and execution scripts use the same shape for `ctx.emit()`:

```javascript
ctx.emit({
  id: "my-op-id",               // optional: auto-generated if omitted
  kind: "js",                    // required: "js" or "http/fetch"
  queue: "site:mysite:js",       // optional: queue key for rate limiting
  dedupKey: "unique-key",        // optional: dedup within the workflow
  input: { ... },                // optional: JSON input for the op
  dependsOn: [                   // optional: dependency list
    { opID: "other-op", required: true }
  ],
  retry: {                       // optional: retry policy
    maxAttempts: 3,
    backoffKind: "exponential",  // "exponential" or "linear"
    initialBackoff: "1s",        // Go duration string or milliseconds
    maxBackoff: "30s",
    multiplier: 2.0
  },
  metadata: {                    // optional: string map
    script: "extract.js"         // required for "js" ops: which script to run
  },
  workflowID: "...",             // optional: override (defaults to current)
  site: "...",                   // optional: override (defaults to current)
  parentID: "..."                // optional: override (defaults to current op)
});
```

The `kind` field determines which runner executes the op:

- `"js"` — runs the script named in `metadata.script` through the JS executor
- `"http/fetch"` — performs an HTTP request based on `input.request`

For `http/fetch` ops, the input shape is:

```javascript
{
  request: {
    method: "GET",
    url: "https://example.com/page"
  },
  persistBody: true,             // save response body as artifact
  artifactName: "page.html"      // artifact display name
}
```

## DependencyResult (ctx.dep)

`ctx.dep(opID)` returns `null` if the dependency doesn't exist, or:

```javascript
{
  opID: "the-op-id",
  data: { ... },                 // decoded result data
  records: [                     // array of written records
    { collection: "items", key: "k", data: { ... } }
  ],
  artifacts: [                   // array of written artifacts
    {
      id: "artifact-id",
      name: "page.html",
      kind: "html",
      contentType: "text/html",
      metadata: { ... },
      bodyText: "<html>..."      // artifact body as string
    }
  ],
  emittedIDs: ["child-op-1"],   // IDs of ops emitted by this op
  completedAt: "2026-...",       // RFC3339Nano timestamp
  error: {                       // present only if the dep failed
    code: "...",
    message: "...",
    retryable: false,
    details: { ... },
    occurredAt: "2026-..."
  }
}
```

## ArtifactSpec (ctx.writeArtifact)

```javascript
ctx.writeArtifact({
  id: "my-artifact",             // optional: auto-generated if omitted
  name: "result.json",           // display name
  kind: "json",                  // artifact kind
  contentType: "application/json",
  metadata: { ... },             // optional string map
  body: "content as string"      // string or bytes
});
```

## Database Modules

Both environments can `require("site-db")` and `require("scraper-db")`:

```javascript
const siteDB = require("site-db");
const scraperDB = require("scraper-db");
```

### Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `db.exec(sql, ...params)` | number | Execute SQL, return affected row count |
| `db.query(sql, ...params)` | array | Execute SQL, return array of row objects |

### Examples

```javascript
// Insert a row
siteDB.exec(
  "INSERT INTO stories (id, title, url) VALUES (?, ?, ?)",
  story.id, story.title, story.url
);

// Query rows
const rows = siteDB.query("SELECT * FROM stories WHERE scraped_at > ?", cutoff);
for (const row of rows) {
  ctx.log("story:", row.title);
}
```

The site DB contains site-specific projection tables defined in `migrations/`. The scraper DB is the engine runtime database — typically used for advanced dedup checks.

## Module Resolution

Execution scripts can `require()` other scripts within the same site's `scripts/` directory:

```javascript
const helpers = require("./lib/common");
```

The resolver tries the path as-is, then appends `.js`. Paths are relative to the site's scripts root.

## Troubleshooting

| Problem | Cause | Solution |
|---------|-------|----------|
| `ctx.dep(opID)` returns null | The dependency hasn't completed or the ID is wrong | Check the emitting script and verify the op ID matches |
| `require("site-db")` throws | The site DB was not opened by the worker | Verify the site is registered and migrations ran |
| `ctx.emit()` throws "kind is required" | The spec object is missing the `kind` field | Always set `kind: "js"` or `kind: "http/fetch"` |
| Script changes are not picked up | The scripts are embedded at compile time | Rebuild with `go run ./cmd/scraper ...` |
| Async script hangs | Promise never resolves | Ensure all code paths resolve or reject |

## See Also

- `scraper help scraper-runtime-model` — Why submit verbs and execution scripts are separate
- `scraper help scraper-architecture-overview` — Broader system map
- `scraper help scraper-adding-a-site` — Step-by-step site-authoring guide

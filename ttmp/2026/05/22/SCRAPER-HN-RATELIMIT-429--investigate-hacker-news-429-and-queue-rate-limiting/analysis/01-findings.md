---
Title: Hacker News 429 and queue rate-limit findings
Ticket: SCRAPER-HN-RATELIMIT-429
Status: active
Topics:
    - scraper
    - scheduler
    - rate-limiting
    - sqlite
    - events
DocType: analysis
Intent: investigation
Owners: []
RelatedFiles:
    - Path: burst
      Note: "1"
    - Path: ratePerSecond
      Note: "1"
    - Path: sites/hackernews/scripts/lib/frontpage.js
      Note: Likely source of malformed query-only pagination URL resolution
    - Path: sites/hackernews/site.yaml
      Note: Defines Hacker News HTTP queue policy maxInFlight=1
    - Path: state/devctl/engine.db
      Note: Engine DB containing failed Hacker News workflows and malformed page-3 fetch ops
    - Path: state/devctl/runtime-events-sessionstream.db
      Note: Runtime event DB proving queue-rate-limited events and >=1s HTTP lease spacing
ExternalSources: []
Summary: Database-backed investigation of Hacker News page-3 429 failures and queue limiter behavior.
LastUpdated: 0001-01-01T00:00:00Z
WhatFor: ""
WhenToUse: ""
---


# Hacker News 429 and queue rate-limit findings

## Goal

Determine from the SQLite databases whether the Hacker News `429 Too Many Requests` failures indicate a broken queue/rate limiter or a different problem.

## Evidence sources

Engine database:

```text
state/devctl/engine.db
```

Sessionstream runtime-event database:

```text
state/devctl/runtime-events-sessionstream.db
```

SQL scripts and captured outputs:

```text
scripts/00_schema.sql
scripts/01_recent_hackernews_failures.sql
scripts/02_http_fetch_timing.sql
scripts/03_queue_state_and_policy.sql
scripts/04_runtime_events_rate_limit.sql
scripts/05_lease_schema_and_rows.sql
scripts/06_http_lease_spacing_from_runtime_events.sql
scripts/07_malformed_url_evidence.sql
sources/00_schema.out
sources/01_recent_hackernews_failures.out
sources/02_http_fetch_timing.out
sources/03_queue_state_and_policy.out
sources/04_runtime_events_rate_limit.out
sources/05_lease_schema_and_rows.out
sources/06_http_lease_spacing_from_runtime_events.out
sources/07_malformed_url_evidence.out
```

## Findings

The most recent Hacker News failures all fail on the same malformed page-3 URL:

```text
https://news.ycombinator.com/?p=2/?p=3
```

`07_malformed_url_evidence.sql` shows five failed `http/fetch` ops with that exact URL and response status `429`. The same query shows that the first page and page 2 succeeded:

```text
https://news.ycombinator.com/          succeeded   5
https://news.ycombinator.com/?p=2      succeeded   5
https://news.ycombinator.com/?p=2/?p=3 failed      5
```

The queue policy exists in `sites/hackernews/site.yaml`:

```yaml
queuePolicies:
  - queue: site:hackernews:http
    maxInFlight: 1
    rateLimit:
      ratePerSecond: 1
      burst: 1
```

The runtime-event DB shows `RUNTIME_EVENT_KIND_QUEUE_RATE_LIMITED` events before the later HTTP fetches. `06_http_lease_spacing_from_runtime_events.sql` computes lease spacing from the runtime-event DB's scheduler `OP_LEASED` events. Every HTTP lease after the first lease in a workflow is at least 1 second after the previous HTTP lease:

```text
1.170s
1.047s
1.158s
1.182s
1.134s
1.144s
1.148s
1.019s
1.146s
1.172s
```

This means the configured queue limiter did run and did enforce the intended approximate 1 request/second start spacing for `site:hackernews:http`.

## Interpretation

The database does not support "the queue limiter failed" as the primary explanation. The better explanation is:

1. The scraper generated a malformed page-3 URL by resolving `?p=3` relative to `https://news.ycombinator.com/?p=2` as if it were a path segment.
2. The HTTP queue limiter still spaced requests at about one request per second.
3. Hacker News returned `429` for the malformed page-3 URL anyway.

The likely source bug is in `sites/hackernews/scripts/lib/frontpage.js`, specifically `toAbsolute(baseURL, href)`. Query-only relative hrefs such as `?p=3` need to resolve against the origin/path, not by string-appending to the current URL.

## Suggested fix direction

Handle query-only hrefs before the generic relative-path append case:

```js
if (value.charAt(0) === "?") {
  return root ? root + "/" + value : value;
}
```

A more robust option is to use URL semantics if available in the goja runtime:

```js
return new URL(value, baseURL).toString();
```

The compatibility of `URL` in the scraper JS runtime should be checked before using it in site scripts.

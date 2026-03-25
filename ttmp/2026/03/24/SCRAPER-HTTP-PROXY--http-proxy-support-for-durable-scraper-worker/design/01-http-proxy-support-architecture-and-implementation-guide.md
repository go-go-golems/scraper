---
Title: HTTP proxy support architecture and implementation guide
Ticket: SCRAPER-HTTP-PROXY
Status: active
Topics:
    - scraper
    - http
    - proxy
    - worker
DocType: design-doc
Intent: long-term
Owners: []
RelatedFiles:
    - Path: pkg/engine/config/config.go
      Note: Current worker HTTP config surface only exposes user agent and timeout
    - Path: pkg/engine/runner/http.go
      Note: HTTP runner construction and request execution path where proxy-aware clients are created
    - Path: pkg/engine/runner/http_test.go
      Note: Existing request-path tests and the right place to add proxy coverage
    - Path: pkg/cmd/worker.go
      Note: Worker flags and config wiring that should expose first-class proxy settings
    - Path: pkg/sites/submitverbs/host.go
      Note: Submission and local runner construction path that should stay aligned with worker HTTP config
ExternalSources: []
Summary: Detailed intern-oriented guide for adding explicit, testable HTTP proxy support to the durable scraper worker and HTTP fetch runner without changing the broader scheduling model.
LastUpdated: 2026-03-24T19:00:00-04:00
WhatFor: Explains the current implicit proxy behavior, the missing first-class config surface, and the recommended phased implementation for explicit worker-level HTTP proxy support.
WhenToUse: Use when implementing, reviewing, or extending proxy support for worker-side http/fetch operations.
---

# HTTP proxy support architecture and implementation guide

## Executive Summary

The scraper worker currently has partial proxy behavior, but only by accident of Go defaults. The `http/fetch` runner constructs an `http.Client` with a timeout and leaves `Transport` as `nil`. In Go, that means the client uses `http.DefaultTransport`, and `http.DefaultTransport` uses `http.ProxyFromEnvironment`. So:

- `HTTP_PROXY` / `HTTPS_PROXY` can already affect worker-side `http/fetch` requests today
- scraper does not document that behavior
- scraper does not expose a first-class worker config or CLI flag for proxy selection
- scraper does not persist or report which proxy policy is being used
- scraper does not have tests proving the behavior

For a new intern, the key conclusion is:

```text
We already have implicit environment-driven proxy support.
We do not yet have explicit scraper-owned proxy support.
```

The goal of this ticket is to add the explicit scraper-owned layer, starting with one global worker-level proxy URL and keeping more advanced routing for later.

## Problem Statement

### Current behavior

The current worker HTTP configuration in [config.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/config/config.go) contains:

- `UserAgent`
- `Timeout`

The HTTP runner in [http.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/http.go) builds:

```go
client = &http.Client{
    Timeout: cfg.Timeout,
}
```

and does not set a custom `Transport`.

That means the worker inherits Go’s default proxy behavior from the process environment. This is useful, but it is not enough for the scraper system because:

- operators cannot see the active proxy setting in CLI flags or config structs
- tests do not prove that proxy routing works
- local smoke runs and worker runs cannot intentionally select a proxy without external environment setup
- future per-site or per-queue proxy policy work has no clean first-class config surface to build on

### Desired first slice

The first slice should be intentionally small and operationally useful:

- add one explicit global proxy URL setting to worker HTTP config
- expose it as `scraper worker run --http-proxy`
- make the HTTP runner construct a dedicated transport when that setting is present
- keep environment proxy behavior as the fallback when no explicit proxy is configured
- add tests for:
  - explicit proxy routing
  - fallback direct routing
  - invalid proxy URL validation

This solves the real operator problem without prematurely designing proxy pools or per-op proxy selection.

## Goals

### Primary goals

- Make proxy behavior explicit in scraper-owned config.
- Keep existing environment-driven behavior as a fallback.
- Add reliable tests for proxy routing through `http/fetch`.
- Keep the first implementation small enough to land quickly.

### Non-goals for the first slice

- proxy authentication UX
- proxy pools
- per-site proxy manifests
- per-op proxy overrides
- SOCKS support unless `net/http` support falls out naturally
- dashboard-level proxy inspection

## Current Architecture

### Request path

Today, worker-side HTTP fetches follow this path:

```text
scraper worker run
  -> build config.HTTP
  -> newDefaultRunnerRegistry(...)
  -> runner.NewHTTPRunner(cfg.HTTP, nil)
  -> http.Client{Timeout: ...}
  -> client.Do(req)
```

Relevant files:

- [worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go)
- [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go)
- [http.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/http.go)

### Why this matters

Both:

- the long-running worker path
- the local submission/local-run helper paths

ultimately rely on `config.HTTP` and `NewHTTPRunner(...)`.

That makes `config.HTTP` the correct configuration seam.

## Recommended Design

## Configuration shape

Extend `config.HTTP` with:

```go
type HTTP struct {
    UserAgent string
    Timeout   time.Duration
    ProxyURL  string
}
```

Rules:

- empty `ProxyURL` means: use Go default environment-driven proxy behavior
- non-empty `ProxyURL` means: force requests through that explicit proxy
- invalid `ProxyURL` should fail fast during config validation or runner construction

This is the simplest contract that gives operators control without removing environment compatibility.

## Worker CLI shape

Expose:

```text
scraper worker run --http-proxy http://127.0.0.1:8081
```

Flag semantics:

- default empty string
- applies to all `http/fetch` ops executed by that worker
- should be shown in `--help`

Optional later follow-up:

- `SCRAPER_HTTP_PROXY` environment variable

That is useful, but not required for the first slice if the flag already exists.

## Runner behavior

When `cfg.ProxyURL == ""`:

- keep current behavior
- use environment fallback through the default transport

When `cfg.ProxyURL != ""`:

- parse URL once
- build a transport with:

```go
transport := http.DefaultTransport.(*http.Transport).Clone()
transport.Proxy = http.ProxyURL(parsedProxyURL)
```

- set that transport on the client

This preserves sensible defaults from the standard transport while making the proxy choice explicit and testable.

## Pseudocode

```text
new worker config
  if http_proxy flag set
    cfg.HTTP.ProxyURL = flag value

new HTTP runner
  if explicit proxy url exists
    parse it
    clone default transport
    set transport.Proxy = ProxyURL(parsed)
    client.Transport = transport
  else
    leave Transport nil
    // Go default transport keeps env proxy support

run op
  client.Do(request)
```

## ASCII Flow Diagram

```text
worker flags/env
  -> config.HTTP
      -> NewHTTPRunner
          -> build http.Client
              -> direct transport or proxied transport
                  -> request execution
```

Decision branch:

```text
ProxyURL empty?
  yes -> use Go default transport behavior
  no  -> use explicit proxied transport
```

## API and CLI references

The intern should read these in order:

1. [config.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/config/config.go)
2. [worker.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/cmd/worker.go)
3. [http.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/http.go)
4. [http_test.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/engine/runner/http_test.go)
5. [host.go](/home/manuel/workspaces/2026-03-23/js-scraper/scraper/pkg/sites/submitverbs/host.go)

## Testing Plan

### Unit/integration tests for the HTTP runner

Add a proxy test with:

- one target server
- one proxy server
- HTTP runner configured with `ProxyURL`
- assert the target is reached through the proxy

A practical test shape:

```text
target server
  records request path
proxy server
  records that it was hit
  forwards to target
runner
  fetches target URL through proxy
assert:
  proxy hit count == 1
  target hit count == 1
```

### Validation tests

- invalid proxy URL should fail
- no explicit proxy should still allow normal direct requests

### CLI help test

Add or extend a root/worker command test so `scraper worker run --help` includes `--http-proxy`.

## Phased Implementation Plan

### Phase 1. Config and runner support

- add `ProxyURL` to `config.HTTP`
- validate obvious malformed values
- add proxied client/transport construction to `NewHTTPRunner`

### Phase 2. Worker and local runner wiring

- add `--http-proxy` to `scraper worker run`
- pass the field into `config.HTTP`
- keep `submitverbs` local runner construction aligned

### Phase 3. Tests

- add HTTP runner proxy test
- add invalid proxy validation test
- add CLI help coverage

### Phase 4. Docs

- update the new ticket diary
- add changelog entry
- optionally add an embedded help page if the implementation is user-facing enough to justify one immediately

## Risks and Sharp Edges

### Implicit environment behavior may surprise reviewers

One source of confusion is that the code already proxies via environment variables even though scraper does not mention proxy support anywhere. The implementation and docs should call that out explicitly so we do not accidentally break it while adding explicit config.

### Shared default transport must not be mutated

Never mutate `http.DefaultTransport` directly. Always clone it before changing `Proxy`.

### Explicit proxy should be deterministic

If `ProxyURL` is set explicitly, the implementation should not also depend on environment variables for proxy choice. The explicit setting should win.

## Recommendation

Build the first slice as:

- explicit worker-level `--http-proxy`
- `config.HTTP.ProxyURL`
- proxied transport in `NewHTTPRunner`
- tests proving routing
- docs clarifying the difference between explicit proxy config and environment fallback

That is enough to solve the current operator need and gives a clean base for future proxy pools or per-op routing.

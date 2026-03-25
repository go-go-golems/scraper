# Tasks

## Analysis and planning

- [x] Create the dedicated `SCRAPER-HTTP-PROXY` ticket workspace
- [x] Inspect the current worker config and HTTP runner
- [x] Identify the gap between implicit env-proxy behavior and explicit scraper-owned proxy config
- [x] Write a detailed architecture and implementation guide
- [x] Record the initial investigation diary
- [x] Upload the ticket bundle to reMarkable

## Implementation

### Phase 1. Config surface

- [x] Extend `pkg/engine/config/config.go` with explicit proxy configuration
- [x] Decide whether the first slice uses a single `ProxyURL` string or a richer struct
- [x] Keep validation strict enough to reject obviously malformed proxy URLs
- [x] Preserve current behavior when no explicit proxy is configured

### Phase 2. HTTP runner transport construction

- [x] Update `pkg/engine/runner/http.go`
- [x] Build a dedicated `http.Transport` when explicit proxy config is set
- [x] Clone `http.DefaultTransport` instead of mutating shared global transport
- [x] Keep environment-based proxy fallback when explicit proxy config is empty
- [x] Make explicit proxy config override environment fallback deterministically

### Phase 3. Worker and local runner wiring

- [x] Add `--http-proxy` to `scraper worker run`
- [x] Pass the flag through `config.HTTP`
- [x] Keep `pkg/sites/submitverbs/host.go` local runner construction aligned with the same config shape
- [x] Decide whether any API/server-side runner construction should also expose the field now or later

### Phase 4. Tests

- [x] Add an HTTP runner test that proves proxy routing
- [x] Add a test for invalid proxy URL handling
- [x] Keep existing direct-fetch tests passing
- [x] Add CLI help coverage for `--http-proxy`
- [x] Run `go test ./... -count=1`

### Phase 5. Documentation and ticket sync

- [x] Update the proxy ticket diary with implementation details and commit hashes
- [x] Update the proxy ticket changelog
- [x] Update the proxy ticket index with implementation status
- [ ] Re-run `docmgr doctor --ticket SCRAPER-HTTP-PROXY --stale-after 30`
- [ ] Upload the refreshed ticket bundle to reMarkable

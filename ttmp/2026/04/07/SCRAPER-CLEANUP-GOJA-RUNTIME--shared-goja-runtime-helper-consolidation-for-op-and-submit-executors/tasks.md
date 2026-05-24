# Tasks

## Analysis And Design

- [x] Create the ticket workspace.
- [x] Identify why the executors should remain separate.
- [x] Identify duplicated helper clusters.
- [x] Write the helper-consolidation design and task breakdown.
- [x] Record the investigation diary.

## Shared Helper Extraction

- [ ] Extract JSON/raw-message helpers into shared code.
- [ ] Extract JS primitive coercion helpers into shared code.
- [ ] Extract map/metadata helpers into shared code.
- [ ] Extract dependency parsing helpers into shared code.
- [ ] Extract retry policy parsing helpers into shared code.
- [ ] Extract module/path helpers where behavior matches.

## Executor Preservation

- [ ] Keep the op executor API stable.
- [ ] Keep the submit executor API stable.
- [ ] Avoid introducing one mode-switched universal executor.
- [ ] Leave context builders and result shaping local unless they can be shared cleanly.

## Validation

- [ ] Run `go test ./pkg/js/runtime -count=1`.
- [ ] Run `go test ./pkg/sites/submitverbs -count=1`.
- [ ] Run `go test ./... -count=1`.
- [ ] Run `docmgr doctor --ticket SCRAPER-CLEANUP-GOJA-RUNTIME --stale-after 30`.

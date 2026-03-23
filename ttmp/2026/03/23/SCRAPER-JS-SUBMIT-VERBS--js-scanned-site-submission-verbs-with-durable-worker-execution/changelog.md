# Changelog

## 2026-03-23

Created the implementation ticket for JS-scanned site submission verbs, wrote the primary design guide, and defined the phased task breakdown for worker-driven durable execution.

Added the first durable worker command as `scraper worker run`, including bounded-cycle smoke-test support, shared runtime helpers for runner and DB setup, and command tests covering help output and empty-engine startup. Commit: `57ffc41`.

Added a new `pkg/sites/submitverbs` host that scans site `verbs/` trees with `go-go-goja/pkg/jsverbs`, mounts discovered commands under `site <site> run`, auto-creates workflows in Go, and lets submission-time JS emit the initial durable ops. Commit: `2ccd9f8`.

Migrated `js-demo` to the new model with `verbs/seed.js`, `verbs/item.js`, and `verbs/summary.js`, removed the old handwritten Go submission command file, added command tests for JS-derived help and submission behavior, and added an end-to-end submit-plus-worker test using the durable engine DB.

Manual smoke validation with the real binary succeeded: `site js-demo run seed` submitted workflow `manual-js-submit`, `engine status` showed one ready op before worker execution and five succeeded ops afterward, and the `demo_runs` site DB row was `manual-js-submit|3|24|224`.

## 2026-03-23

- Initial workspace created

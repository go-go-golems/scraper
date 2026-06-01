# Tasks

## TODO

- [x] Add bump-go-go-golems make target (from go-template)
- [x] Add GOWORK=off prefix to all make targets that need it (align with go-template)
- [x] Bump go-go-golems dependencies: glazed 1.2.14→latest, go-go-goja 0.4.16→latest, sessionstream 0.0.5→latest
- [x] Fix compilation after dependency bumps (API changes)
- [x] Update CI workflows to match go-template (add release.yaml with split build, update push.yml with logcopter-check and generated file verification)
- [x] Update lefthook.yml to use make lint instead of make lintmax gosec govulncheck (align with go-template)
- [x] Add LICENSE file (MIT, from go-template)
- [x] Verify build, tests, and lint pass after all changes
- [x] Update version tag (svu patch) and push to publish

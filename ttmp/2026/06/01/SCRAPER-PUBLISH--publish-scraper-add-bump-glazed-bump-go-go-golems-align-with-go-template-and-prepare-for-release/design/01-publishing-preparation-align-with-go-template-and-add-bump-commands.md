---
Title: 'Publishing preparation: align with go-template and add bump commands'
Ticket: SCRAPER-PUBLISH
Status: active
Topics:
    - release
    - go
    - glazed
    - scraper
DocType: design
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-template/.github/workflows/release.yaml
      Note: Reference for split linux/darwin release workflow
    - Path: ../../../../../../../../../../code/wesen/go-go-golems/go-template/Makefile
      Note: Reference template with bump-go-go-golems and GOWORK=off patterns
    - Path: needs GOWORK
      Note: off alignment
    - Path: scraper/.github/workflows/push.yml
      Note: Needs logcopter-check and generated file verification
    - Path: scraper/Makefile
      Note: Missing bump-go-go-golems target
    - Path: scraper/go.mod
      Note: Stale go-go-golems dependencies (glazed v1.2.14
ExternalSources: []
Summary: Prepare scraper for its first real publish by aligning Makefile targets with go-template, bumping stale go-go-golems dependencies, updating CI workflows, and verifying the full build/lint/test cycle.
LastUpdated: 2026-06-01T16:36:18.270524953-04:00
WhatFor: Guide the implementation of all changes needed to publish scraper as a proper go-go-golems release.
WhenToUse: Reference when bumping deps, adding Makefile targets, or updating CI for scraper.
---


# Publishing Preparation: Align with go-template and Add Bump Commands

## Executive Summary

Scraper is tagged at `v0.0.1` but depends on severely outdated go-go-golems packages (glazed `v1.2.14` vs latest `v1.3.6`, go-go-goja `v0.4.16` vs latest `v0.7.2`). The Makefile lacks the standard `bump-go-go-golems` target from go-template, CI workflows diverge from the template pattern, and the project has no LICENSE file. This doc maps all gaps and provides a phased plan to make scraper publishable.

## Problem Statement

1. **Stale dependencies**: glazed is ~20 patch versions behind, go-go-goja is ~30 versions behind. This means scraper is missing bug fixes, API improvements, and possibly incompatible changes.
2. **No bump workflow**: The Makefile has no `bump-go-go-golems` target, making dependency updates manual and error-prone.
3. **CI divergence**: Scraper's CI uses a monolithic `push.yml` without logcopter-check or generated-file verification. The release workflow uses a single-job approach instead of the split linux/darwin pattern from go-template.
4. **Missing LICENSE**: The project has no LICENSE file (go-template uses MIT).
5. **GOWORK interference**: The project is in a go.work workspace, so many make targets need `GOWORK=off` when run standalone (the template uses this consistently, scraper does not).

## Gap Analysis: Scraper vs go-template

### 1. Makefile

| Feature | go-template | scraper | Status |
|---------|-------------|---------|--------|
| `bump-go-go-golems` target | ✅ Auto-bumps all go-go-golems deps | ❌ Missing | **Needs adding** |
| `GOWORK=off` on all targets | ✅ Consistent | ❌ Only on some (goreleaser, release) | **Needs fixing** |
| `tag-major/minor/patch` | ✅ | ✅ | OK |
| `release` target | ✅ With `GOWORK=off` | ✅ Without `GOWORK=off` | **Needs fixing** |
| `logcopter-generate/check` | ✅ With `GOWORK=off` | ✅ With `GOWORK=off` | OK |
| `install` target | ✅ Simple build+copy | ✅ build-go then copy | OK (different but functional) |

**Proposed `bump-go-go-golems` target** (from go-template, adapted):

```makefile
bump-go-go-golems:
	@deps="$$(awk '/^require[[:space:]]+github\.com\/go-go-golems\// { print $$2 } /^[[:space:]]*github\.com\/go-go-golems\// { print $$1 }' go.mod | sort -u)"; \
	if [ -z "$$deps" ]; then \
		echo "No github.com/go-go-golems dependencies in go.mod"; \
	else \
		echo "Bumping go-go-golems dependencies:"; \
		echo "$$deps"; \
		for dep in $$deps; do GOWORK=off go get "$${dep}@latest"; done; \
	fi
	GOWORK=off go mod tidy
```

### 2. CI Workflows

| Workflow | go-template | scraper | Status |
|----------|-------------|---------|--------|
| `push.yml` | Logcopter-check + generate + git diff --exit-code + test | Only generate + test | **Needs updating** |
| `release.yaml` | Split linux/darwin + merge + GPG sign | Missing (no release.yaml) | **Needs adding** |
| `lint.yml` | Same pattern | Same pattern | OK |
| `dependency-scanning.yml` | gosec with direct install | gosec via `make gosec` | Minor difference |
| `secret-scanning.yml` | Same | Same | OK |
| `codeql-analysis.yml` | Same | Same | OK |

**Key CI updates needed**:
- Add `logcopter-check` step to push.yml
- Add `git diff --exit-code` step to verify generated files are committed
- Add `release.yaml` with the split linux/darwin build pattern
- Fix gosec workflow to install directly (not via make)

### 3. Dependency Versions (current vs latest)

| Dependency | Current | Latest | Delta |
|------------|---------|--------|-------|
| glazed | v1.2.14 | v1.3.6 | ~20 versions |
| go-go-goja | v0.4.16 | v0.7.2 | ~30 versions |
| sessionstream | v0.0.5 | v0.0.6 | 1 version |
| logcopter | v0.1.0 | v0.1.0 | Up to date |

### 4. Other Files

| File | go-template | scraper | Status |
|------|-------------|---------|--------|
| LICENSE | MIT | Missing | **Needs adding** |
| .gitignore | Standard | Extended (good) | OK |
| lefthook.yml | `make lint` + `make test` | `make lintmax gosec govulncheck` + `make test` | Heavier than template, but OK |

## Phased Implementation Plan

### Phase 1: Makefile alignment (low risk)

1. Add `bump-go-go-golems` target to Makefile
2. Add `GOWORK=off` to `release` target
3. Ensure all make targets that touch go modules use `GOWORK=off`

### Phase 2: Dependency bump (medium risk — may break compilation)

1. Run `make bump-go-go-golems`
2. Fix compilation errors from API changes
3. Run tests, fix failures

### Phase 3: CI alignment (low risk)

1. Update `push.yml` with logcopter-check and generated file verification
2. Add `release.yaml` from template
3. Fix `dependency-scanning.yml` gosec invocation

### Phase 4: Housekeeping (low risk)

1. Add LICENSE (MIT)
2. Verify full build/lint/test cycle
3. Tag and push

## Risks

1. **go-go-goja API changes**: The jump from v0.4.16 to v0.7.2 may include breaking changes to the module system or engine initialization. This is the highest-risk change.
2. **glazed API changes**: Staticcheck exclusions in scraper's `.golangci.yml` reference `cli.CreateProcessorLegacy` and `gggengine.DefaultRegistryModules` — these may have been removed or renamed.
3. **GOWORK vs standalone**: The scraper is part of a go.work workspace. Bumping deps with `GOWORK=off` will use proxy.golang.org, which may resolve differently from the local workspace replacements.

## References

- go-template repo: `~/code/wesen/go-go-golems/go-template/`
- scraper Makefile: `scraper/Makefile`
- scraper go.mod: `scraper/go.mod`
- scraper CI: `scraper/.github/workflows/`

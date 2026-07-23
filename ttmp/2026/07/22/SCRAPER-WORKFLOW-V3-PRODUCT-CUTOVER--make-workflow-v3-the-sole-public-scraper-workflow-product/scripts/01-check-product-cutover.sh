#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

new_surface=(
  pkg/workflowv3product
  pkg/taskpackages
  pkg/cmd/workflow_v3.go
  examples/workflowv3
)
legacy_import='"github\.com/go-go-golems/scraper/pkg/(engine|workflow|services|sites|js/runtime)(/|")'

if rg -n "$legacy_import" "${new_surface[@]}"; then
  echo "ERROR: Workflow V3 product surface imports a superseded package" >&2
  exit 1
fi

GOWORK=off go test ./pkg/workflowv3product ./pkg/cmd \
  -run 'Test(Product|WorkflowV3CLI|RootWorkflow)' -count=1

help=$(GOWORK=off go run ./cmd/scraper worker run --help)
grep -q -- '--workflow-db' <<<"$help"
grep -q -- '--artifact-root' <<<"$help"
if grep -q -- '--engine-db' <<<"$help"; then
  echo "ERROR: public worker command still exposes the legacy engine" >&2
  exit 1
fi

legacy_help=$(GOWORK=off go run ./cmd/scraper legacy worker run --help)
grep -q -- '--engine-db' <<<"$legacy_help"

# The old engine cannot be deleted while the repository still has production
# callers. This is an explicit guard, not approval for new dependencies.
legacy_callers=$( (rg -l "$legacy_import" pkg/api pkg/cmd pkg/js pkg/services pkg/sites pkg/workflow 2>/dev/null || true) | wc -l)
if (( legacy_callers == 0 )); then
  echo "ERROR: legacy deletion gate changed; update the ticket inventory before deleting" >&2
  exit 1
fi

if [[ -n "${RAG_EVAL_REPO:-}" ]]; then
  if [[ ! -d "$RAG_EVAL_REPO" ]]; then
    echo "ERROR: RAG_EVAL_REPO is not a directory: $RAG_EVAL_REPO" >&2
    exit 1
  fi
  downstream=$( (rg -l "$legacy_import" "$RAG_EVAL_REPO" --glob '*.go' 2>/dev/null || true) | wc -l)
  if (( downstream == 0 )); then
    echo "ERROR: downstream deletion gate changed; update the ticket inventory" >&2
    exit 1
  fi
  printf 'downstream legacy callers: %d\n' "$downstream"
fi

printf 'Workflow V3 product cutover guard passed; local legacy callers: %d\n' "$legacy_callers"

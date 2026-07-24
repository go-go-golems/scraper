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

for removed in legacy engine api site; do
  if GOWORK=off go run ./cmd/scraper "$removed" --help >/dev/null 2>&1; then
    echo "ERROR: removed command remains available: $removed" >&2
    exit 1
  fi
done

legacy_callers=$( (rg -l "$legacy_import" --glob '*.go' --glob '!ttmp/**' . 2>/dev/null || true) | wc -l)
if (( legacy_callers != 0 )); then
  echo "ERROR: $legacy_callers local legacy importers remain" >&2
  exit 1
fi

if [[ -n "${RAG_EVAL_REPO:-}" ]]; then
  if [[ ! -d "$RAG_EVAL_REPO" ]]; then
    echo "ERROR: RAG_EVAL_REPO is not a directory: $RAG_EVAL_REPO" >&2
    exit 1
  fi
  downstream=$( (rg -l "$legacy_import" "$RAG_EVAL_REPO" --glob '*.go' 2>/dev/null || true) | wc -l)
  if (( downstream != 0 )); then
    echo "ERROR: $downstream downstream legacy importers remain" >&2
    exit 1
  fi
  printf 'downstream legacy callers: 0\n'
fi

printf 'Workflow V3 sole-product guard passed; local legacy callers: 0\n'

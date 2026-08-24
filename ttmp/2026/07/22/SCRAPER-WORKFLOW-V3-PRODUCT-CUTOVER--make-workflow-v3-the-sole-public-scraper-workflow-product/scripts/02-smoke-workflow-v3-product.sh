#!/usr/bin/env bash
set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"
[[ -x dist/scraper ]] || { echo "dist/scraper is missing; run make build-go" >&2; exit 1; }

root=$(mktemp -d)
worker_session="scraper-v3-worker-$$"
api_session="scraper-v3-api-$$"
api_port=${SCRAPER_V3_SMOKE_PORT:-18991}
cleanup() {
  tmux kill-session -t "$worker_session" 2>/dev/null || true
  tmux kill-session -t "$api_session" 2>/dev/null || true
  rm -rf "$root"
}
trap cleanup EXIT
cp examples/workflowv3/cookbook-linear/* "$root/"
common=(--workflow-db "$root/workflow.db" --artifact-root "$root/artifacts" --poll-interval 10ms)

./dist/scraper workflow validate "$root/workflow.js" >"$root/validate.json"
./dist/scraper workflow compile "$root/workflow.js" --out "$root/plan.json"
./dist/scraper workflow "${common[@]}" submit "$root/workflow.js" --inputs "$root/inputs.json" --run-id smoke-restart >"$root/submit.json"

tmux new-session -d -s "$worker_session" \
  "$repo_root/dist/scraper worker --workflow-db '$root/workflow.db' --artifact-root '$root/artifacts' --poll-interval 10ms run >'$root/worker.log' 2>&1"

status=""
for _ in $(seq 1 300); do
  ./dist/scraper workflow "${common[@]}" runs show smoke-restart >"$root/show.json"
  status=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["snapshot"]["status"])' "$root/show.json")
  [[ "$status" == "succeeded" ]] && break
  sleep 0.02
done
[[ "$status" == "succeeded" ]] || { echo "worker did not finish: $status" >&2; exit 1; }
tmux kill-session -t "$worker_session"
./dist/scraper workflow "${common[@]}" runs follow smoke-restart >"$root/follow.ndjson"

./dist/scraper workflow "${common[@]}" submit "$root/workflow.js" --inputs "$root/inputs.json" --run-id smoke-api-cancel >"$root/api-submit.json"
export SCRAPER_WORKFLOW_OPERATOR_TOKEN=smoke-operator-token
tmux new-session -d -s "$api_session" \
  "SCRAPER_WORKFLOW_OPERATOR_TOKEN='$SCRAPER_WORKFLOW_OPERATOR_TOKEN' '$repo_root/dist/scraper' workflow --workflow-db '$root/workflow.db' --artifact-root '$root/artifacts' serve --address '127.0.0.1:$api_port' >'$root/api.log' 2>&1"
for _ in $(seq 1 100); do
  curl -fsS "http://127.0.0.1:$api_port/api/v3/workflow/health" >"$root/health.json" 2>/dev/null && break
  sleep 0.02
done
curl -fsS "http://127.0.0.1:$api_port/api/v3/workflow/runs" >"$root/api-runs.json"
unauthorized=$(curl -sS -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:$api_port/api/v3/workflow/runs/smoke-api-cancel/cancel")
[[ "$unauthorized" == "403" ]]
curl -fsS -X POST -H "Authorization: Bearer $SCRAPER_WORKFLOW_OPERATOR_TOKEN" \
  "http://127.0.0.1:$api_port/api/v3/workflow/runs/smoke-api-cancel/cancel" >"$root/cancel.json"
tmux kill-session -t "$api_session"

python3 - "$root" <<'PY'
import json, pathlib, sys
root = pathlib.Path(sys.argv[1])
validate = json.loads((root / "validate.json").read_text())
show = json.loads((root / "show.json").read_text())
canceled = json.loads((root / "cancel.json").read_text())
print(json.dumps({
    "validationOK": validate["ok"],
    "planDigest": validate["planDigest"],
    "restartRunStatus": show["snapshot"]["status"],
    "restartRunAttempts": len(show["snapshot"]["attempts"]),
    "followLines": len((root / "follow.ndjson").read_text().splitlines()),
    "unauthorizedCancelStatus": 403,
    "authorizedCancelStatus": canceled["snapshot"]["status"],
}, indent=2, sort_keys=True))
PY

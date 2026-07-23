#!/usr/bin/env bash
set -euo pipefail

scraper_root=$(git rev-parse --show-toplevel)
research_root=${RESEARCHCTL_REPO:-/home/manuel/workspaces/2026-06-30/benchmark-cpu-inference/researchctl}
[[ -d "$scraper_root" ]] || { echo "SCRAPER_REPO is not a directory" >&2; exit 1; }
work=$(mktemp -d)
cleanup() {
  if [[ "${KEEP_SMOKE_WORK:-0}" == "1" ]]; then
    echo "preserved smoke work: $work" >&2
  else
    rm -rf "$work"
  fi
}
trap cleanup EXIT
mkdir -p "$work/bin"

(cd "$research_root" && GOWORK=off go build -o "$work/bin/researchctl" ./cmd/researchctl)
(cd "$scraper_root" && GOWORK=off go build -o "$work/bin/scraper" ./cmd/scraper)
(cd "$scraper_root" && GOWORK=off go build -o "$work/bin/scraper-workflow-runner" ./cmd/scraper-workflow-runner)

"$work/bin/scraper" workflow --task-package research-runner-fixture researchctl-config \
  "$scraper_root/examples/research-runner/workflow.js" \
  --bindings "$scraper_root/examples/research-runner/bindings.json" \
  --out "$work/generated-execution.json"
"$work/bin/researchctl" experiment explain-plan "$research_root/examples/lab/scraper-workflow-plan.js" --output json > "$work/explain.json"
python3 - "$work/generated-execution.json" "$work/explain.json" <<'PY'
import json,sys
generated=json.load(open(sys.argv[1]))
embedded=json.load(open(sys.argv[2]))['schedule'][0]['specification']['canonicalIdentity']['domainConfig']
assert generated == embedded, 'embedded Researchctl domain config is stale'
PY

cp "$research_root/examples/lab/scraper-workflow-project.js" "$work/project.js"
cp "$research_root/examples/lab/scraper-workflow-plan.js" "$work/plan.js"
"$work/bin/researchctl" lab init --project "$work/project.js" --database "$work/lab.db" >/dev/null
mkdir -p "$work/artifacts/inputs"
cp "$scraper_root/examples/research-runner/input-a.json" "$work/artifacts/inputs/scraper-runner-a.json"
cp "$scraper_root/examples/research-runner/input-b.json" "$work/artifacts/inputs/scraper-runner-b.json"

cat > "$work/bin/crash-once-runner" <<EOF
#!/usr/bin/env python3
import json, os, subprocess, sys
try:
    os.mkdir("$work/crash-claimed")
except FileExistsError:
    os.execv("$work/bin/scraper-workflow-runner", ["$work/bin/scraper-workflow-runner", *sys.argv[1:]])
request = sys.stdin.buffer.read()
child = subprocess.Popen(["$work/bin/scraper-workflow-runner", *sys.argv[1:]], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
child.stdin.write(request)
child.stdin.close()
for line in child.stdout:
    sys.stdout.buffer.write(line)
    sys.stdout.buffer.flush()
    frame = json.loads(line)
    if frame.get("type") == "event" and frame.get("event", {}).get("type") == "workflow.submitted":
        child.kill()
        child.wait()
        sys.exit(17)
sys.exit(child.wait())
EOF
chmod +x "$work/bin/crash-once-runner"

run_args=(
  experiment run-plan "$work/plan.js"
  --project "$work/project.js" --database "$work/lab.db"
  --runner-command "$work/bin/crash-once-runner"
  --runner-name scraper-workflow-runner --runner-version v1
  --runner-arg=--state-root --runner-arg="$work/scraper-state"
  --runner-arg=--artifact-root --runner-arg="$work/scraper-artifacts"
  --runner-arg=--poll-interval --runner-arg=2ms
  --max-attempts 2 --timeout 20s --output json
)
"$work/bin/researchctl" "${run_args[@]}" > "$work/result.json"
"$work/bin/researchctl" "${run_args[@]}" > "$work/resume.json"

# Extract one canonical specification for a cancellation/timeout run.
python3 - "$work/explain.json" "$work/cancel-spec.json" <<'PY'
import json,sys
x=json.load(open(sys.argv[1]))
json.dump(x['schedule'][0]['specification'],open(sys.argv[2],'w'),sort_keys=True,separators=(',',':'))
PY
"$work/bin/researchctl" lab init --project "$work/project.js" --database "$work/cancel-lab.db" >/dev/null
mkdir -p "$work/cancel-artifacts/inputs"
cp "$scraper_root/examples/research-runner/input-a.json" "$work/cancel-artifacts/inputs/scraper-runner-a.json"
set +e
"$work/bin/researchctl" experiment execute-spec "$work/cancel-spec.json" \
  --project "$work/project.js" --database "$work/cancel-lab.db" --experiment-id EXP-SCRAPER-WORKFLOW \
  --runner-command "$work/bin/scraper-workflow-runner" \
  --runner-name scraper-workflow-runner --runner-version v1 \
  --runner-arg=--state-root --runner-arg="$work/cancel-state" \
  --runner-arg=--artifact-root --runner-arg="$work/cancel-scraper-artifacts" \
  --runner-arg=--capacity --runner-arg=blocked.resource=1 \
  --runner-arg=--poll-interval --runner-arg=2ms \
  --runner-arg=--cancellation-timeout --runner-arg=2s \
  --runner-cancel-grace 3s --max-attempts 1 --timeout 300ms --output json \
  > "$work/cancel-result.json" 2> "$work/cancel-stderr.txt"
cancel_exit=$?
set -e
[[ $cancel_exit -ne 0 ]] || { echo "timeout execution unexpectedly succeeded" >&2; exit 1; }

# Re-project every terminal subordinate database in a fresh Scraper process.
mkdir -p "$work/reprojected"
for database in "$work"/scraper-state/*.db "$work"/cancel-state/*.db; do
  read -r run_id status < <(python3 - "$database" <<'PY'
import sqlite3,sys
row=sqlite3.connect(sys.argv[1]).execute('select run_id,status from v3_runs').fetchone()
print(row[0],row[1])
PY
)
  if [[ "$status" == "succeeded" || "$status" == "failed" || "$status" == "canceled" ]]; then
    "$work/bin/scraper" workflow --workflow-db "$database" \
      --artifact-root "$work/reproject-artifacts" \
      --task-package research-runner-fixture observations "$run_id" \
      > "$work/reprojected/$run_id.json"
  fi
done

python3 - "$work" <<'PY'
import glob,json,os,sqlite3,sys
root=sys.argv[1]
result=json.load(open(root+'/result.json'))
resume=json.load(open(root+'/resume.json'))
assert (result['scheduled'],result['executed'],result['resumed'],result['failed']) == (4,4,0,0)
assert (resume['scheduled'],resume['executed'],resume['resumed'],resume['failed']) == (4,0,4,0)
attempt_counts=[]
for item in result['items']:
    execution=item['execution']
    assert execution['status']=='succeeded'
    attempt_counts.append(len(execution['attempts']))
    selected=execution['export']['attempts'][-1]
    metrics={m['name']:m.get('numericProjection') for m in selected['metrics']}
    assert metrics['workflow.retries']==1
    assert metrics['workflow.external_operations.failed']==1
    assert len(selected['artifacts'])==4
    assert all(a['verification']['status']=='verified' for a in selected['artifacts'])
    artifact=next(a for a in selected['artifacts'] if a['kind']=='scraper-workflow-observations')
    observation=json.load(open(os.path.join(root,'artifacts',artifact['uri'])))
    assert observation['schemaVersion']=='scraper-workflow-observations/v1'
    assert observation['derivationVersion']=='workflow-observations/v1'
    assert observation['privacyClass']=='bounded-identifiers-digests-integers'
    assert observation['digest']==artifact['metadata']['digest']
    assert observation['sourceDigest']==artifact['metadata']['sourceDigest']
    metric_by_name={m['name']:m for m in observation['metrics']}
    assert metric_by_name['workflow.retries']['value']==1
    assert metric_by_name['workflow.external_operations.failed']['value']==1
    assert metric_by_name['workflow.external_operations.elapsed_sum']['value']==2000
    assert metric_by_name['workflow.external_operations.elapsed_union']['value']==2000
    assert metric_by_name['workflow.external_operations.completion_coverage']['value']=={'numerator':2,'denominator':2}
    assert len(observation['artifactLineage'])==1
    assert sorted(t['kind'] for t in observation['traces'])==['workflow.artifact_lineage','workflow.critical_path','workflow.failures']
    assert {m['name'] for m in selected['metrics']}==set(metric_by_name)
    for metric in selected['metrics']:
        assert metric['metadata']['sourceDigest']==observation['sourceDigest']
        assert metric['metadata']['observationDigest']==observation['digest']
    reprojected=json.load(open(os.path.join(root,'reprojected',observation['runId']+'.json')))
    assert reprojected==observation
assert sorted(attempt_counts)==[1,1,1,2],attempt_counts
workflow_databases=glob.glob(root+'/scraper-state/*.db')
assert len(workflow_databases)==5,workflow_databases
cancel=json.load(open(root+'/cancel-result.json'))
assert cancel['status']=='failed',cancel['status']
terminal=cancel['export']['attempts'][0]['terminalSummary']['payload']
assert terminal['kind']=='timeout',terminal
databases=glob.glob(root+'/cancel-state/*.db')
assert len(databases)==1,databases
connection=sqlite3.connect(databases[0])
workflow_run_id,status=connection.execute('select run_id,status from v3_runs').fetchone()
assert status=='canceled',status
canceled_observation=json.load(open(os.path.join(root,'reprojected',workflow_run_id+'.json')))
assert canceled_observation['runStatus']=='canceled'
assert next(m for m in canceled_observation['metrics'] if m['name']=='workflow.job_attempts')['value']==0
summary={
  'matrix': {'scheduled':4,'executed':4,'researchAttemptCounts':sorted(attempt_counts),'workflowRunsIncludingCrashedAttempt':len(workflow_databases)},
  'workflowPerAttempt': {'retries':1,'failedExternalOperations':1,'artifacts':4},
  'resume': {'executed':0,'resumed':4},
  'observations': {'runnerDomainSchemaVersion':'scraper-workflow-execution/v2','schemaVersion':'scraper-workflow-observations/v1','derivationVersion':'workflow-observations/v1','metricsPerRun':22,'tracesPerRun':3,'retryAwareOperationElapsedMicros':2000,'restartReprojectionMatched':True},
  'timeout': {'researchStatus':cancel['status'],'researchFailureKind':terminal['kind'],'workflowStatus':status,'durableCanceledObservation':True},
  'contractFixtureMatched': True,
}
print(json.dumps(summary,indent=2,sort_keys=True))
PY

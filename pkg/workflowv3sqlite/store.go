package workflowv3sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaSQL string

var ErrStaleCompletion = errors.New("workflow v3 stale completion")

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("workflow v3 SQLite path is required")
	}
	uri := url.URL{Scheme: "file", Path: path}
	query := uri.Query()
	query.Set("_foreign_keys", "on")
	query.Set("_journal_mode", "WAL")
	query.Set("_busy_timeout", "5000")
	query.Set("_txlock", "immediate")
	uri.RawQuery = query.Encode()
	dsn := uri.String()
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open workflow v3 SQLite: %w", err)
	}
	db.SetMaxOpenConns(4)
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate workflow v3 SQLite: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) CreateRun(ctx context.Context, runID workflowv3.RunID, plan workflowv3.WorkflowPlan, inputs map[string]workflowv3.ArtifactRef, now time.Time) error {
	if strings.TrimSpace(string(runID)) == "" {
		return fmt.Errorf("run ID is required")
	}
	if err := validatePlanDigest(plan); err != nil {
		return err
	}
	if err := validateRunInputs(plan, inputs); err != nil {
		return err
	}
	planBody, err := workflowv3.CanonicalJSON(plan)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stamp := formatTime(now)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_runs(run_id, name, plan_digest, plan_json, status, created_at, updated_at)
VALUES (?, ?, ?, ?, 'running', ?, ?)`, runID, plan.Name, plan.Digest, planBody, stamp, stamp); err != nil {
		return fmt.Errorf("insert workflow v3 run: %w", err)
	}
	for _, input := range plan.Inputs {
		ref := inputs[input.Name]
		if err := insertRef(ctx, tx, `
INSERT INTO v3_run_inputs(run_id, name, schema_id, digest, media_type, size_bytes, locator)
VALUES (?, ?, ?, ?, ?, ?, ?)`, []any{runID, input.Name}, ref); err != nil {
			return fmt.Errorf("insert run input %s: %w", input.Name, err)
		}
	}
	for ordinal, node := range plan.Nodes {
		bindings, _ := workflowv3.CanonicalJSON(node.Bindings)
		inputSchemas, _ := workflowv3.CanonicalJSON(node.InputSchemas)
		outputSchemas, _ := workflowv3.CanonicalJSON(node.OutputSchemas)
		modules, _ := workflowv3.CanonicalJSON(node.Modules)
		identity := node.Implementation
		if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_nodes(
  run_id, node_key, ordinal, task_kind, task_version, bundle_digest,
  entrypoint, task_abi, bindings_json, input_schemas_json,
  output_schemas_json, modules_json, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
			runID, node.Key, ordinal, identity.Kind, identity.Version,
			identity.BundleDigest, identity.Entrypoint, identity.ABI,
			bindings, inputSchemas, outputSchemas, modules,
		); err != nil {
			return fmt.Errorf("insert node %s: %w", node.Key, err)
		}
	}
	for _, node := range plan.Nodes {
		for _, dependency := range node.DependsOn {
			if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_dependencies(run_id, node_key, dependency_key)
VALUES (?, ?, ?)`, runID, node.Key, dependency); err != nil {
				return fmt.Errorf("insert dependency %s -> %s: %w", node.Key, dependency, err)
			}
		}
	}
	if err := insertEvent(ctx, tx, runID, "", "run.created", map[string]any{
		"planDigest": plan.Digest,
	}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) LeaseNext(ctx context.Context, registry *workflowv3.SealedRegistry, now time.Time, duration time.Duration) (*workflowv3.Lease, error) {
	if registry == nil {
		return nil, fmt.Errorf("sealed registry is required")
	}
	if duration <= 0 {
		return nil, fmt.Errorf("lease duration must be positive")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := reclaimExpired(ctx, tx, now); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `
SELECT
  n.run_id, n.node_key, n.task_kind, n.task_version, n.bundle_digest,
  n.entrypoint, n.task_abi, n.bindings_json, n.input_schemas_json,
  n.output_schemas_json, n.modules_json, n.attempt_count, r.cancel_epoch
FROM v3_nodes n
JOIN v3_runs r ON r.run_id = n.run_id
WHERE n.status = 'pending' AND r.status = 'running'
  AND NOT EXISTS (
    SELECT 1
    FROM v3_dependencies d
    JOIN v3_nodes dependency
      ON dependency.run_id = d.run_id
     AND dependency.node_key = d.dependency_key
    WHERE d.run_id = n.run_id
      AND d.node_key = n.node_key
      AND dependency.status != 'succeeded'
  )
ORDER BY r.created_at, n.ordinal, n.run_id
LIMIT 100`)
	if err != nil {
		return nil, fmt.Errorf("query ready nodes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var selected *leaseCandidate
	for rows.Next() {
		candidate, err := scanLeaseCandidate(rows)
		if err != nil {
			return nil, err
		}
		if _, err := registry.Resolve(candidate.node.Implementation); err == nil {
			selected = &candidate
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if selected == nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	attempt := selected.attemptCount + 1
	token := uuid.NewString()
	expires := now.Add(duration)
	result, err := tx.ExecContext(ctx, `
UPDATE v3_nodes
SET status = 'running', attempt_count = ?, lease_token = ?,
    lease_cancel_epoch = ?, lease_expires_at = ?
WHERE run_id = ? AND node_key = ? AND status = 'pending'`,
		attempt, token, selected.cancelEpoch, formatTime(expires),
		selected.runID, selected.node.Key,
	)
	if err != nil {
		return nil, fmt.Errorf("lease node: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, ErrStaleCompletion
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_attempts(
  run_id, node_key, attempt_no, status, lease_token, cancel_epoch,
  registry_generation, started_at
) VALUES (?, ?, ?, 'running', ?, ?, ?, ?)`,
		selected.runID, selected.node.Key, attempt, token, selected.cancelEpoch,
		registry.Generation(), formatTime(now),
	); err != nil {
		return nil, fmt.Errorf("insert attempt: %w", err)
	}
	if err := insertEvent(ctx, tx, selected.runID, selected.node.Key, "attempt.started", map[string]any{
		"attempt": attempt, "registryGeneration": registry.Generation(),
	}, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &workflowv3.Lease{
		RunID: selected.runID, NodeKey: selected.node.Key, Attempt: attempt,
		Token: token, CancelEpoch: selected.cancelEpoch, ExpiresAt: expires,
		PlanNode: selected.node, RegistryGeneration: registry.Generation(),
	}, nil
}

func (s *Store) ResolveInputs(ctx context.Context, lease workflowv3.Lease) (map[string]workflowv3.ArtifactRef, error) {
	ret := make(map[string]workflowv3.ArtifactRef, len(lease.PlanNode.Bindings))
	for port, binding := range lease.PlanNode.Bindings {
		var row *sql.Row
		switch binding.Source {
		case "input":
			row = s.db.QueryRowContext(ctx, `
SELECT schema_id, digest, media_type, size_bytes, locator
FROM v3_run_inputs WHERE run_id = ? AND name = ?`, lease.RunID, binding.Name)
		case "node-output":
			row = s.db.QueryRowContext(ctx, `
SELECT schema_id, digest, media_type, size_bytes, locator
FROM v3_node_outputs WHERE run_id = ? AND node_key = ? AND port = ?`,
				lease.RunID, binding.NodeKey, binding.Port)
		default:
			return nil, fmt.Errorf("unsupported binding source %q", binding.Source)
		}
		ref, err := scanRef(row)
		if err != nil {
			return nil, fmt.Errorf("resolve input %s: %w", port, err)
		}
		if ref.Schema != lease.PlanNode.InputSchemas[port] {
			return nil, fmt.Errorf("resolved input %s schema %q does not match %q", port, ref.Schema, lease.PlanNode.InputSchemas[port])
		}
		ret[port] = ref
	}
	return ret, nil
}

func (s *Store) Complete(ctx context.Context, lease workflowv3.Lease, outputs map[string]workflowv3.ArtifactRef, now time.Time) error {
	if err := validateOutputs(lease.PlanNode, outputs); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := checkFence(ctx, tx, lease); err != nil {
		return err
	}
	ports := make([]string, 0, len(outputs))
	for port := range outputs {
		ports = append(ports, port)
	}
	sort.Strings(ports)
	for _, port := range ports {
		if err := insertRef(ctx, tx, `
INSERT INTO v3_node_outputs(
  run_id, node_key, port, schema_id, digest, media_type, size_bytes, locator
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, []any{lease.RunID, lease.NodeKey, port}, outputs[port]); err != nil {
			return fmt.Errorf("insert output %s: %w", port, err)
		}
	}
	stamp := formatTime(now)
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_attempts SET status = 'succeeded', finished_at = ?
WHERE run_id = ? AND node_key = ? AND attempt_no = ? AND status = 'running'`,
		stamp, lease.RunID, lease.NodeKey, lease.Attempt); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_nodes
SET status = 'succeeded', lease_token = NULL, lease_cancel_epoch = NULL,
    lease_expires_at = NULL
WHERE run_id = ? AND node_key = ? AND lease_token = ? AND status = 'running'`,
		lease.RunID, lease.NodeKey, lease.Token); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_runs SET status = 'succeeded', updated_at = ?
WHERE run_id = ? AND status = 'running'
  AND NOT EXISTS (
    SELECT 1 FROM v3_nodes WHERE run_id = ? AND status != 'succeeded'
  )`, stamp, lease.RunID, lease.RunID); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, lease.RunID, lease.NodeKey, "node.succeeded", map[string]any{
		"attempt": lease.Attempt, "outputs": outputDigests(outputs),
	}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Fail(ctx context.Context, lease workflowv3.Lease, failure workflowv3.Failure, now time.Time) error {
	if err := workflowv3.ValidateFailure(failure); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := checkFence(ctx, tx, lease); err != nil {
		return err
	}
	stamp := formatTime(now)
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_attempts
SET status = 'failed', finished_at = ?, failure_class = ?, failure_code = ?,
    failure_retryable = ?, failure_message = ?
WHERE run_id = ? AND node_key = ? AND attempt_no = ? AND status = 'running'`,
		stamp, failure.Class, failure.Code, failure.Retryable, failure.Message,
		lease.RunID, lease.NodeKey, lease.Attempt); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_nodes SET status = 'failed', lease_token = NULL,
  lease_cancel_epoch = NULL, lease_expires_at = NULL
WHERE run_id = ? AND node_key = ? AND lease_token = ? AND status = 'running'`,
		lease.RunID, lease.NodeKey, lease.Token); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_runs SET status = 'failed', updated_at = ?
WHERE run_id = ? AND status = 'running'`, stamp, lease.RunID); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, lease.RunID, lease.NodeKey, "node.failed", map[string]any{
		"attempt": lease.Attempt, "class": failure.Class,
		"code": failure.Code, "retryable": failure.Retryable,
	}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Cancel(ctx context.Context, runID workflowv3.RunID, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stamp := formatTime(now)
	result, err := tx.ExecContext(ctx, `
UPDATE v3_runs SET status = 'canceled', cancel_epoch = cancel_epoch + 1,
  updated_at = ? WHERE run_id = ? AND status = 'running'`, stamp, runID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("run %s is not running", runID)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_attempts SET status = 'canceled', finished_at = ?
WHERE run_id = ? AND status = 'running'`, stamp, runID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_nodes SET status = 'canceled', lease_token = NULL,
  lease_cancel_epoch = NULL, lease_expires_at = NULL
WHERE run_id = ? AND status IN ('pending', 'running')`, runID); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, runID, "", "run.canceled", map[string]any{}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) Snapshot(ctx context.Context, runID workflowv3.RunID) (workflowv3.RunSnapshot, error) {
	var snapshot workflowv3.RunSnapshot
	snapshot.RunID = runID
	var planBody []byte
	if err := s.db.QueryRowContext(ctx, `
SELECT status, plan_digest, plan_json FROM v3_runs WHERE run_id = ?`, runID).
		Scan(&snapshot.Status, &snapshot.PlanDigest, &planBody); err != nil {
		return workflowv3.RunSnapshot{}, err
	}
	var plan workflowv3.WorkflowPlan
	if err := workflowv3.StrictDecode(planBody, &plan); err != nil {
		return workflowv3.RunSnapshot{}, err
	}
	snapshot.Outputs = map[string]workflowv3.ArtifactRef{}
	for _, output := range plan.Outputs {
		var row *sql.Row
		switch output.Value.Source {
		case "input":
			row = s.db.QueryRowContext(ctx, `
SELECT schema_id, digest, media_type, size_bytes, locator
FROM v3_run_inputs WHERE run_id = ? AND name = ?`, runID, output.Value.Name)
		case "node-output":
			row = s.db.QueryRowContext(ctx, `
SELECT schema_id, digest, media_type, size_bytes, locator
FROM v3_node_outputs WHERE run_id = ? AND node_key = ? AND port = ?`,
				runID, output.Value.NodeKey, output.Value.Port)
		default:
			return workflowv3.RunSnapshot{}, fmt.Errorf("unsupported output source %q", output.Value.Source)
		}
		ref, err := scanRef(row)
		if err == sql.ErrNoRows && snapshot.Status != "succeeded" {
			continue
		}
		if err != nil {
			return workflowv3.RunSnapshot{}, fmt.Errorf("resolve run output %s: %w", output.Name, err)
		}
		snapshot.Outputs[output.Name] = ref
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT node_key, attempt_no, status, cancel_epoch, registry_generation,
  started_at, finished_at, failure_class, failure_code,
  failure_retryable, failure_message
FROM v3_attempts WHERE run_id = ? ORDER BY node_key, attempt_no`, runID)
	if err != nil {
		return workflowv3.RunSnapshot{}, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		attempt, err := scanAttempt(rows, runID)
		if err != nil {
			return workflowv3.RunSnapshot{}, err
		}
		snapshot.Attempts = append(snapshot.Attempts, attempt)
	}
	return snapshot, rows.Err()
}

func (s *Store) Checkpoint(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

type leaseCandidate struct {
	runID        workflowv3.RunID
	node         workflowv3.PlanNode
	attemptCount int
	cancelEpoch  int64
}

type rowScanner interface {
	Scan(...any) error
}

func scanLeaseCandidate(rows rowScanner) (leaseCandidate, error) {
	var candidate leaseCandidate
	var kind, version, bundleDigest, entrypoint, abi string
	var bindings, inputSchemas, outputSchemas, modules []byte
	if err := rows.Scan(
		&candidate.runID, &candidate.node.Key, &kind, &version, &bundleDigest,
		&entrypoint, &abi, &bindings, &inputSchemas, &outputSchemas, &modules,
		&candidate.attemptCount, &candidate.cancelEpoch,
	); err != nil {
		return candidate, err
	}
	candidate.node.Implementation = workflowv3.ImplementationIdentity{
		TaskKey:      workflowv3.TaskKey{Kind: kind, Version: version},
		BundleDigest: bundleDigest, Entrypoint: entrypoint, ABI: abi,
	}
	if err := workflowv3.StrictDecode(bindings, &candidate.node.Bindings); err != nil {
		return candidate, err
	}
	if err := workflowv3.StrictDecode(inputSchemas, &candidate.node.InputSchemas); err != nil {
		return candidate, err
	}
	if err := workflowv3.StrictDecode(outputSchemas, &candidate.node.OutputSchemas); err != nil {
		return candidate, err
	}
	if err := workflowv3.StrictDecode(modules, &candidate.node.Modules); err != nil {
		return candidate, err
	}
	return candidate, nil
}

func reclaimExpired(ctx context.Context, tx *sql.Tx, now time.Time) error {
	stamp := formatTime(now)
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_attempts SET status = 'lease_lost', finished_at = ?
WHERE status = 'running' AND EXISTS (
  SELECT 1 FROM v3_nodes n
  WHERE n.run_id = v3_attempts.run_id
    AND n.node_key = v3_attempts.node_key
    AND n.lease_expires_at < ?
)`, stamp, stamp); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
UPDATE v3_nodes SET status = 'pending', lease_token = NULL,
  lease_cancel_epoch = NULL, lease_expires_at = NULL
WHERE status = 'running' AND lease_expires_at < ?`, stamp)
	return err
}

func checkFence(ctx context.Context, tx *sql.Tx, lease workflowv3.Lease) error {
	var status string
	var token sql.NullString
	var cancelEpoch int64
	var leaseCancelEpoch sql.NullInt64
	err := tx.QueryRowContext(ctx, `
SELECT r.status, r.cancel_epoch, n.lease_token, n.lease_cancel_epoch
FROM v3_runs r JOIN v3_nodes n ON n.run_id = r.run_id
WHERE r.run_id = ? AND n.node_key = ?`, lease.RunID, lease.NodeKey).
		Scan(&status, &cancelEpoch, &token, &leaseCancelEpoch)
	if err != nil {
		return err
	}
	if status != "running" || !token.Valid || token.String != lease.Token ||
		cancelEpoch != lease.CancelEpoch || !leaseCancelEpoch.Valid ||
		leaseCancelEpoch.Int64 != lease.CancelEpoch {
		return ErrStaleCompletion
	}
	return nil
}

func validatePlanDigest(plan workflowv3.WorkflowPlan) error {
	if plan.Schema != workflowv3.PlanSchema || strings.TrimSpace(plan.Digest) == "" {
		return fmt.Errorf("valid workflow v3 plan and digest are required")
	}
	withoutDigest := plan
	withoutDigest.Digest = ""
	digest, err := workflowv3.Digest(withoutDigest)
	if err != nil {
		return err
	}
	if digest != plan.Digest {
		return fmt.Errorf("plan digest mismatch: got %s want %s", plan.Digest, digest)
	}
	return nil
}

func validateRunInputs(plan workflowv3.WorkflowPlan, inputs map[string]workflowv3.ArtifactRef) error {
	if len(inputs) != len(plan.Inputs) {
		return fmt.Errorf("run has %d inputs, plan requires %d", len(inputs), len(plan.Inputs))
	}
	for _, input := range plan.Inputs {
		ref, ok := inputs[input.Name]
		if !ok {
			return fmt.Errorf("run input %q is missing", input.Name)
		}
		if err := workflowv3.ValidateArtifactRef(ref); err != nil {
			return fmt.Errorf("run input %q: %w", input.Name, err)
		}
		if ref.Schema != input.Schema {
			return fmt.Errorf("run input %q schema %q does not match %q", input.Name, ref.Schema, input.Schema)
		}
	}
	return nil
}

func validateOutputs(node workflowv3.PlanNode, outputs map[string]workflowv3.ArtifactRef) error {
	if len(outputs) != len(node.OutputSchemas) {
		return fmt.Errorf("node %s has %d outputs, expected %d", node.Key, len(outputs), len(node.OutputSchemas))
	}
	for port, schema := range node.OutputSchemas {
		ref, ok := outputs[port]
		if !ok {
			return fmt.Errorf("node %s output %q is missing", node.Key, port)
		}
		if err := workflowv3.ValidateArtifactRef(ref); err != nil {
			return fmt.Errorf("node %s output %q: %w", node.Key, port, err)
		}
		if ref.Schema != schema {
			return fmt.Errorf("node %s output %q schema %q does not match %q", node.Key, port, ref.Schema, schema)
		}
	}
	return nil
}

func insertRef(ctx context.Context, tx *sql.Tx, statement string, prefix []any, ref workflowv3.ArtifactRef) error {
	args := append(append([]any(nil), prefix...), ref.Schema, ref.Digest, ref.MediaType, ref.Size, ref.Locator)
	_, err := tx.ExecContext(ctx, statement, args...)
	return err
}

func scanRef(row rowScanner) (workflowv3.ArtifactRef, error) {
	var ref workflowv3.ArtifactRef
	err := row.Scan(&ref.Schema, &ref.Digest, &ref.MediaType, &ref.Size, &ref.Locator)
	return ref, err
}

func insertEvent(ctx context.Context, tx *sql.Tx, runID workflowv3.RunID, nodeKey workflowv3.NodeKey, eventType string, payload any, now time.Time) error {
	body, err := workflowv3.CanonicalJSON(payload)
	if err != nil {
		return err
	}
	var node any
	if nodeKey != "" {
		node = nodeKey
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO v3_events(run_id, node_key, event_type, payload_json, created_at)
VALUES (?, ?, ?, ?, ?)`, runID, node, eventType, body, formatTime(now))
	return err
}

func outputDigests(outputs map[string]workflowv3.ArtifactRef) map[string]string {
	ret := make(map[string]string, len(outputs))
	for port, ref := range outputs {
		ret[port] = ref.Digest
	}
	return ret
}

func scanAttempt(row rowScanner, runID workflowv3.RunID) (workflowv3.Attempt, error) {
	var attempt workflowv3.Attempt
	attempt.RunID = runID
	var started, finished sql.NullString
	var failureClass, failureCode, failureMessage sql.NullString
	var failureRetryable sql.NullBool
	if err := row.Scan(
		&attempt.NodeKey, &attempt.Number, &attempt.Status, &attempt.CancelEpoch,
		&attempt.RegistryGeneration, &started, &finished, &failureClass,
		&failureCode, &failureRetryable, &failureMessage,
	); err != nil {
		return attempt, err
	}
	attempt.StartedAt, _ = time.Parse(time.RFC3339Nano, started.String)
	if finished.Valid {
		attempt.FinishedAt, _ = time.Parse(time.RFC3339Nano, finished.String)
	}
	if failureClass.Valid {
		attempt.Failure = &workflowv3.Failure{
			Class: failureClass.String, Code: failureCode.String,
			Retryable: failureRetryable.Bool, Message: failureMessage.String,
		}
	}
	return attempt, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

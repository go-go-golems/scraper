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
	if err := migrateSliceThreeToFive(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate workflow v3 slice 3-5 columns: %w", err)
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
	for _, input := range plan.SetInputs {
		ref := inputs[input.Name]
		if err := insertRef(ctx, tx, `
INSERT INTO v3_run_inputs(run_id, name, schema_id, digest, media_type, size_bytes, locator)
VALUES (?, ?, ?, ?, ?, ?, ?)`, []any{runID, input.Name}, ref); err != nil {
			return fmt.Errorf("insert run set input %s: %w", input.Name, err)
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
  output_schemas_json, modules_json, resource_class, max_attempts,
  retry_backoff_ms, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
			runID, node.Key, ordinal, identity.Kind, identity.Version,
			identity.BundleDigest, identity.Entrypoint, identity.ABI,
			bindings, inputSchemas, outputSchemas, modules,
			node.ResourceClass, node.Retry.MaxAttempts, node.Retry.BackoffMillis,
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
	for _, mapped := range plan.Maps {
		var source *workflowv3.ArtifactRef
		if mapped.Source.Source == "set-input" {
			ref := inputs[mapped.Source.Name]
			source = &ref
		}
		var sourceSchema, sourceDigest, sourceMediaType, sourceLocator any
		var sourceSize any
		if source != nil {
			sourceSchema, sourceDigest = source.Schema, source.Digest
			sourceMediaType, sourceSize, sourceLocator = source.MediaType, source.Size, source.Locator
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_expansions(
  run_id, map_key, source_schema, source_digest, source_media_type,
  source_size_bytes, source_locator, page_size, max_items,
  max_materialized_ahead, status, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)`,
			runID, mapped.Key, sourceSchema, sourceDigest, sourceMediaType,
			sourceSize, sourceLocator, mapped.Policy.PageSize, mapped.Policy.MaxItems,
			mapped.Policy.MaxMaterializedAhead, stamp); err != nil {
			return fmt.Errorf("insert expansion %s: %w", mapped.Key, err)
		}
	}
	for _, reduced := range plan.Reductions {
		var source *workflowv3.ArtifactRef
		var sourceName, sourceMapKey any
		switch reduced.Source.Source {
		case "set-input":
			ref := inputs[reduced.Source.Name]
			source = &ref
			sourceName = reduced.Source.Name
		case "map-output":
			sourceMapKey = reduced.Source.MapKey
		}
		var sourceSchema, sourceDigest, sourceMediaType, sourceSize, sourceLocator any
		if source != nil {
			sourceSchema, sourceDigest = source.Schema, source.Digest
			sourceMediaType, sourceSize, sourceLocator = source.MediaType, source.Size, source.Locator
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_reductions(
  run_id, reduce_key, source_kind, source_name, source_map_key,
  source_schema, source_digest, source_media_type, source_size_bytes,
  source_locator, fan_in, max_levels, status, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?)`,
			runID, reduced.Key, reduced.Source.Source, sourceName, sourceMapKey,
			sourceSchema, sourceDigest, sourceMediaType, sourceSize, sourceLocator,
			reduced.Policy.FanIn, reduced.Policy.MaxLevels, stamp); err != nil {
			return fmt.Errorf("insert reduction %s: %w", reduced.Key, err)
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
	return s.LeaseNextWithResources(ctx, registry, nil, now, duration)
}

// LeaseNextWithResources atomically admits one ready node whose resource class
// has database-scoped capacity. A nil capacity map is unlimited and exists for
// deterministic single-step execution; dispatchers always provide capacities.
func (s *Store) LeaseNextWithResources(
	ctx context.Context,
	registry *workflowv3.SealedRegistry,
	capacities map[string]int,
	now time.Time,
	duration time.Duration,
) (*workflowv3.Lease, error) {
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
	active, err := activeResources(ctx, tx)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `
SELECT
  n.run_id, n.node_key, n.task_kind, n.task_version, n.bundle_digest,
  n.entrypoint, n.task_abi, n.bindings_json, n.input_schemas_json,
  n.output_schemas_json, n.modules_json, n.resource_class,
  n.max_attempts, n.retry_backoff_ms, n.attempt_count, r.cancel_epoch
FROM v3_nodes n
JOIN v3_runs r ON r.run_id = n.run_id
LEFT JOIN v3_run_resource_dispatch fairness
  ON fairness.run_id = n.run_id
 AND fairness.resource_class = n.resource_class
WHERE n.status = 'pending' AND r.status = 'running'
  AND (n.ready_at IS NULL OR julianday(n.ready_at) <= julianday(?))
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
ORDER BY COALESCE(fairness.dispatch_count, 0), r.created_at, n.ordinal, n.run_id`,
		formatTime(now))
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
		if _, err := registry.ResolveNode(candidate.node); err != nil {
			continue
		}
		if capacities != nil {
			capacity, configured := capacities[candidate.node.ResourceClass]
			if !configured || capacity < 1 || active[candidate.node.ResourceClass] >= capacity {
				continue
			}
		}
		selected = &candidate
		break
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
  registry_generation, resource_class, started_at
) VALUES (?, ?, ?, 'running', ?, ?, ?, ?, ?)`,
		selected.runID, selected.node.Key, attempt, token, selected.cancelEpoch,
		registry.Generation(), selected.node.ResourceClass, formatTime(now),
	); err != nil {
		return nil, fmt.Errorf("insert attempt: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_run_resource_dispatch(run_id, resource_class, dispatch_count)
VALUES (?, ?, 1)
ON CONFLICT(run_id, resource_class)
DO UPDATE SET dispatch_count = dispatch_count + 1`,
		selected.runID, selected.node.ResourceClass); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_runs SET updated_at = ? WHERE run_id = ?`,
		formatTime(now), selected.runID); err != nil {
		return nil, err
	}
	if err := insertEvent(ctx, tx, selected.runID, selected.node.Key, "attempt.started", map[string]any{
		"attempt": attempt, "registryGeneration": registry.Generation(),
		"resourceClass": selected.node.ResourceClass,
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

func (s *Store) LeaseValid(ctx context.Context, lease workflowv3.Lease, now time.Time) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM v3_runs r JOIN v3_nodes n ON n.run_id = r.run_id
WHERE r.run_id = ? AND n.node_key = ?
  AND r.status = 'running' AND r.cancel_epoch = ?
  AND n.status = 'running' AND n.lease_token = ?
  AND n.lease_cancel_epoch = ?
  AND julianday(n.lease_expires_at) >= julianday(?)`,
		lease.RunID,
		lease.NodeKey,
		lease.CancelEpoch,
		lease.Token,
		lease.CancelEpoch,
		formatTime(now),
	).Scan(&count)
	return count == 1, err
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
		case "map-item":
			row = s.db.QueryRowContext(ctx, `
SELECT input_schema, input_digest, input_media_type, input_size_bytes, input_locator
FROM v3_map_items WHERE run_id = ? AND map_key = ? AND node_key = ?`,
				lease.RunID, binding.MapKey, lease.NodeKey)
		case "reduction-partition":
			row = s.db.QueryRowContext(ctx, `
SELECT input_schema, input_digest, input_media_type, input_size_bytes, input_locator
FROM v3_reduction_partitions
WHERE run_id = ? AND reduce_key = ? AND node_key = ?`,
				lease.RunID, binding.ReduceKey, lease.NodeKey)
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
    lease_expires_at = NULL, ready_at = NULL
WHERE run_id = ? AND node_key = ? AND lease_token = ? AND status = 'running'`,
		lease.RunID, lease.NodeKey, lease.Token); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_expansions
SET terminal_items = terminal_items + 1,
    status = CASE
      WHEN status = 'expanded' AND terminal_items + 1 = total_items
      THEN 'succeeded' ELSE status END,
    updated_at = ?
WHERE run_id = ? AND EXISTS (
  SELECT 1 FROM v3_map_items item
  WHERE item.run_id = v3_expansions.run_id
    AND item.map_key = v3_expansions.map_key
    AND item.node_key = ?
)`, stamp, lease.RunID, lease.NodeKey); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_reduction_partitions SET status = 'succeeded'
WHERE run_id = ? AND node_key = ? AND status IN ('pending','running')`,
		lease.RunID, lease.NodeKey); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_runs SET status = 'succeeded', updated_at = ?
WHERE run_id = ? AND status = 'running'
  AND NOT EXISTS (
    SELECT 1 FROM v3_nodes WHERE run_id = ? AND status != 'succeeded'
  )
  AND NOT EXISTS (
    SELECT 1 FROM v3_expansions WHERE run_id = ? AND status != 'published'
  )
  AND NOT EXISTS (
    SELECT 1 FROM v3_reductions WHERE run_id = ? AND status != 'published'
  )`, stamp, lease.RunID, lease.RunID, lease.RunID, lease.RunID); err != nil {
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
	retry := failure.Retryable && lease.Attempt < lease.PlanNode.Retry.MaxAttempts
	if retry {
		readyAt := now.Add(time.Duration(lease.PlanNode.Retry.BackoffMillis) * time.Millisecond)
		if _, err := tx.ExecContext(ctx, `
UPDATE v3_nodes SET status = 'pending', lease_token = NULL,
  lease_cancel_epoch = NULL, lease_expires_at = NULL, ready_at = ?
WHERE run_id = ? AND node_key = ? AND lease_token = ? AND status = 'running'`,
			formatTime(readyAt), lease.RunID, lease.NodeKey, lease.Token); err != nil {
			return err
		}
		if err := insertEvent(ctx, tx, lease.RunID, lease.NodeKey, "node.retry_scheduled", map[string]any{
			"attempt": lease.Attempt, "class": failure.Class,
			"code": failure.Code, "readyAt": formatTime(readyAt),
		}, now); err != nil {
			return err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `
UPDATE v3_nodes SET status = 'failed', lease_token = NULL,
  lease_cancel_epoch = NULL, lease_expires_at = NULL
WHERE run_id = ? AND node_key = ? AND lease_token = ? AND status = 'running'`,
			lease.RunID, lease.NodeKey, lease.Token); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE v3_expansions SET status = 'failed', updated_at = ?
WHERE run_id = ? AND status NOT IN ('published','failed','canceled')
  AND EXISTS (
    SELECT 1 FROM v3_map_items item
    WHERE item.run_id = v3_expansions.run_id
      AND item.map_key = v3_expansions.map_key
      AND item.node_key = ?
  )`, stamp, lease.RunID, lease.NodeKey); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE v3_reduction_partitions SET status = 'failed'
WHERE run_id = ? AND node_key = ? AND status IN ('pending','running')`,
			lease.RunID, lease.NodeKey); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE v3_reductions SET status = 'failed', updated_at = ?
WHERE run_id = ? AND status NOT IN ('published','failed','canceled')
  AND EXISTS (
    SELECT 1 FROM v3_reduction_partitions partition
    WHERE partition.run_id = v3_reductions.run_id
      AND partition.reduce_key = v3_reductions.reduce_key
      AND partition.node_key = ?
  )`, stamp, lease.RunID, lease.NodeKey); err != nil {
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
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_expansions SET status = 'canceled', updated_at = ?
WHERE run_id = ? AND status != 'published'`, stamp, runID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_reductions SET status = 'canceled', updated_at = ?
WHERE run_id = ? AND status != 'published'`, stamp, runID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_reduction_partitions SET status = 'canceled'
WHERE run_id = ? AND status IN ('pending','running')`, runID); err != nil {
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
		case "reduction-output":
			row = s.db.QueryRowContext(ctx, `
SELECT root_schema, root_digest, root_media_type, root_size_bytes, root_locator
FROM v3_reductions WHERE run_id = ? AND reduce_key = ? AND status = 'published'`,
				runID, output.Value.ReduceKey)
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
	for _, output := range plan.SetOutputs {
		if output.Value.Source != "map-output" {
			return workflowv3.RunSnapshot{}, fmt.Errorf("unsupported set output source %q", output.Value.Source)
		}
		row := s.db.QueryRowContext(ctx, `
SELECT output_schema, output_digest, output_media_type, output_size_bytes,
  output_locator
FROM v3_expansions WHERE run_id = ? AND map_key = ? AND status = 'published'`,
			runID, output.Value.MapKey)
		ref, err := scanRef(row)
		if err == sql.ErrNoRows && snapshot.Status != "succeeded" {
			continue
		}
		if err != nil {
			return workflowv3.RunSnapshot{}, fmt.Errorf("resolve run set output %s: %w", output.Name, err)
		}
		snapshot.Outputs[output.Name] = ref
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT node_key, attempt_no, status, cancel_epoch, registry_generation,
  resource_class, started_at, finished_at, failure_class, failure_code,
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

func activeResources(ctx context.Context, tx *sql.Tx) (map[string]int, error) {
	rows, err := tx.QueryContext(ctx, `
SELECT resource_class, COUNT(*) FROM v3_nodes
WHERE status = 'running' GROUP BY resource_class`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	active := map[string]int{}
	for rows.Next() {
		var resource string
		var count int
		if err := rows.Scan(&resource, &count); err != nil {
			return nil, err
		}
		active[resource] = count
	}
	return active, rows.Err()
}

func scanLeaseCandidate(rows rowScanner) (leaseCandidate, error) {
	var candidate leaseCandidate
	var kind, version, bundleDigest, entrypoint, abi string
	var bindings, inputSchemas, outputSchemas, modules []byte
	if err := rows.Scan(
		&candidate.runID, &candidate.node.Key, &kind, &version, &bundleDigest,
		&entrypoint, &abi, &bindings, &inputSchemas, &outputSchemas, &modules,
		&candidate.node.ResourceClass, &candidate.node.Retry.MaxAttempts,
		&candidate.node.Retry.BackoffMillis, &candidate.attemptCount,
		&candidate.cancelEpoch,
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
    AND julianday(n.lease_expires_at) < julianday(?)
)`, stamp, stamp); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
UPDATE v3_nodes SET status = 'pending', lease_token = NULL,
  lease_cancel_epoch = NULL, lease_expires_at = NULL
WHERE status = 'running'
  AND julianday(lease_expires_at) < julianday(?)`, stamp)
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
	required := len(plan.Inputs) + len(plan.SetInputs)
	if len(inputs) != required {
		return fmt.Errorf("run has %d inputs, plan requires %d", len(inputs), required)
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
	for _, input := range plan.SetInputs {
		ref, ok := inputs[input.Name]
		if !ok {
			return fmt.Errorf("run set input %q is missing", input.Name)
		}
		if err := workflowv3.ValidateArtifactRef(ref); err != nil {
			return fmt.Errorf("run set input %q: %w", input.Name, err)
		}
		if ref.Schema != input.ManifestSchema {
			return fmt.Errorf("run set input %q schema %q does not match %q", input.Name, ref.Schema, input.ManifestSchema)
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
		&attempt.RegistryGeneration, &attempt.ResourceClass, &started, &finished, &failureClass,
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

func migrateSliceThreeToFive(ctx context.Context, db *sql.DB) error {
	migrations := []struct {
		table, column, definition string
	}{
		{"v3_runs", "dispatch_count", "INTEGER NOT NULL DEFAULT 0"},
		{"v3_nodes", "resource_class", "TEXT NOT NULL DEFAULT 'cpu.default'"},
		{"v3_nodes", "max_attempts", "INTEGER NOT NULL DEFAULT 1"},
		{"v3_nodes", "retry_backoff_ms", "INTEGER NOT NULL DEFAULT 0"},
		{"v3_nodes", "ready_at", "TEXT"},
		{"v3_attempts", "resource_class", "TEXT NOT NULL DEFAULT 'cpu.default'"},
	}
	for _, migration := range migrations {
		exists, err := columnExists(ctx, db, migration.table, migration.column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		statement := fmt.Sprintf(
			"ALTER TABLE %s ADD COLUMN %s %s",
			migration.table,
			migration.column,
			migration.definition,
		)
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func columnExists(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

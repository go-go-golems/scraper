package workflowv3sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type ReductionCandidate struct {
	RunID        workflowv3.RunID
	ReduceKey    string
	Status       string
	CurrentLevel int
	FanIn        int
	MaxLevels    int
	SourceItems  int
	Source       workflowv3.ArtifactRef
}

type ReductionPartitionInput struct {
	Partition workflowv3.ReductionPartition
	Ref       workflowv3.ArtifactRef
}

func (s *Store) ReductionCandidates(ctx context.Context) ([]ReductionCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT reduction.run_id, reduction.reduce_key, reduction.status,
  reduction.current_level, reduction.fan_in, reduction.max_levels,
  reduction.source_items,
  COALESCE(reduction.source_schema, expansion.output_schema),
  COALESCE(reduction.source_digest, expansion.output_digest),
  COALESCE(reduction.source_media_type, expansion.output_media_type),
  COALESCE(reduction.source_size_bytes, expansion.output_size_bytes),
  COALESCE(reduction.source_locator, expansion.output_locator)
FROM v3_reductions reduction
JOIN v3_runs run ON run.run_id = reduction.run_id
LEFT JOIN v3_expansions expansion
  ON expansion.run_id = reduction.run_id
 AND expansion.map_key = reduction.source_map_key
 AND expansion.status = 'published'
WHERE run.status = 'running' AND reduction.status IN ('pending','executing')
  AND COALESCE(reduction.source_digest, expansion.output_digest) IS NOT NULL
ORDER BY run.created_at, reduction.reduce_key`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var candidates []ReductionCandidate
	for rows.Next() {
		var candidate ReductionCandidate
		if err := rows.Scan(
			&candidate.RunID, &candidate.ReduceKey, &candidate.Status,
			&candidate.CurrentLevel, &candidate.FanIn, &candidate.MaxLevels,
			&candidate.SourceItems, &candidate.Source.Schema, &candidate.Source.Digest,
			&candidate.Source.MediaType, &candidate.Source.Size,
			&candidate.Source.Locator,
		); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func (s *Store) ReductionLevelMembers(
	ctx context.Context,
	runID workflowv3.RunID,
	reduceKey string,
) ([]workflowv3.ManifestItem, string, int, bool, error) {
	_, reduced, _, err := s.loadPlanReduction(ctx, runID, reduceKey)
	if err != nil {
		return nil, "", 0, false, err
	}
	var status, sourceDigest string
	var currentLevel, currentItems int
	if err := s.db.QueryRowContext(ctx, `
SELECT status, source_digest, current_level, current_items
FROM v3_reductions WHERE run_id = ? AND reduce_key = ?`, runID, reduceKey).Scan(
		&status, &sourceDigest, &currentLevel, &currentItems,
	); err != nil {
		return nil, "", 0, false, err
	}
	if status != "executing" {
		return nil, sourceDigest, currentLevel + 1, false, nil
	}
	var terminal int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM v3_reduction_partitions
WHERE run_id = ? AND reduce_key = ? AND level = ? AND status = 'succeeded'`,
		runID, reduceKey, currentLevel).Scan(&terminal); err != nil {
		return nil, "", 0, false, err
	}
	if terminal != currentItems {
		return nil, sourceDigest, currentLevel + 1, false, nil
	}
	if len(reduced.OutputSchemas) != 1 {
		return nil, "", 0, false, fmt.Errorf("reduction %q must have one output", reduceKey)
	}
	var outputPort string
	for port := range reduced.OutputSchemas {
		outputPort = port
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT partition.ordinal, output.schema_id, output.digest, output.media_type,
  output.size_bytes, output.locator
FROM v3_reduction_partitions partition
JOIN v3_node_outputs output
  ON output.run_id = partition.run_id AND output.node_key = partition.node_key
WHERE partition.run_id = ? AND partition.reduce_key = ?
  AND partition.level = ? AND partition.status = 'succeeded'
  AND output.port = ?
ORDER BY partition.ordinal`, runID, reduceKey, currentLevel, outputPort)
	if err != nil {
		return nil, "", 0, false, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]workflowv3.ManifestItem, 0, currentItems)
	for rows.Next() {
		var ordinal int
		var item workflowv3.ManifestItem
		if err := rows.Scan(
			&ordinal, &item.Value.Schema, &item.Value.Digest, &item.Value.MediaType,
			&item.Value.Size, &item.Value.Locator,
		); err != nil {
			return nil, "", 0, false, err
		}
		item.Key = fmt.Sprintf("level-%04d-partition-%08d", currentLevel, ordinal)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", 0, false, err
	}
	if len(items) != currentItems {
		return nil, "", 0, false, fmt.Errorf("reduction %q level %d output cardinality mismatch", reduceKey, currentLevel)
	}
	return items, sourceDigest, currentLevel + 1, true, nil
}

func (s *Store) MaterializeReductionLevel(
	ctx context.Context,
	runID workflowv3.RunID,
	reduceKey string,
	source workflowv3.ArtifactRef,
	sourceItems int,
	level int,
	partitions []ReductionPartitionInput,
	now time.Time,
) error {
	if err := workflowv3.ValidateArtifactRef(source); err != nil {
		return err
	}
	_, reduced, reduceOrdinal, err := s.loadPlanReduction(ctx, runID, reduceKey)
	if err != nil {
		return err
	}
	if sourceItems < 2 || len(partitions) < 1 || level < 0 || level >= reduced.Policy.MaxLevels {
		return fmt.Errorf("invalid reduction level materialization")
	}
	for ordinal, input := range partitions {
		if err := workflowv3.ValidateReductionPartition(input.Partition, reduced.Policy.FanIn); err != nil {
			return err
		}
		body, err := workflowv3.EncodeReductionPartition(input.Partition, reduced.Policy.FanIn)
		if err != nil {
			return err
		}
		digest, err := workflowv3.Digest(input.Partition)
		if err != nil {
			return err
		}
		if input.Partition.ReduceKey != reduceKey || input.Partition.SourceDigest != source.Digest ||
			input.Partition.Level != level || input.Partition.Ordinal != ordinal ||
			input.Ref.Schema != workflowv3.ReductionPartitionSchemaV1 ||
			input.Ref.Digest != digest || input.Ref.Size != int64(len(body)) {
			return fmt.Errorf("reduction partition %d identity mismatch", ordinal)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var runStatus, status string
	var currentLevel, existingSourceItems int
	var existingDigest sql.NullString
	if err := tx.QueryRowContext(ctx, `
SELECT run.status, reduction.status, reduction.current_level,
  reduction.source_items, reduction.source_digest
FROM v3_reductions reduction JOIN v3_runs run ON run.run_id = reduction.run_id
WHERE reduction.run_id = ? AND reduction.reduce_key = ?`, runID, reduceKey).Scan(
		&runStatus, &status, &currentLevel, &existingSourceItems, &existingDigest,
	); err != nil {
		return err
	}
	if runStatus != "running" || (status != "pending" && status != "executing") {
		return fmt.Errorf("reduction %s/%s is not materializable", runID, reduceKey)
	}
	if status == "pending" {
		if level != 0 {
			return fmt.Errorf("first reduction level must be zero")
		}
	} else {
		if !existingDigest.Valid || existingDigest.String != source.Digest ||
			existingSourceItems != sourceItems {
			return fmt.Errorf("reduction level source changed")
		}
		if level == currentLevel {
			rows, err := tx.QueryContext(ctx, `
SELECT ordinal, input_digest FROM v3_reduction_partitions
WHERE run_id = ? AND reduce_key = ? AND level = ? ORDER BY ordinal`,
				runID, reduceKey, level)
			if err != nil {
				return err
			}
			matched := 0
			for rows.Next() {
				var ordinal int
				var digest string
				if err := rows.Scan(&ordinal, &digest); err != nil {
					_ = rows.Close()
					return err
				}
				if ordinal >= len(partitions) || partitions[ordinal].Ref.Digest != digest {
					_ = rows.Close()
					return fmt.Errorf("reduction level conflicts with existing partition")
				}
				matched++
			}
			if err := rows.Err(); err != nil {
				_ = rows.Close()
				return err
			}
			if err := rows.Close(); err != nil {
				return err
			}
			if matched != len(partitions) {
				return fmt.Errorf("reduction level conflicts with existing partition count")
			}
			return tx.Commit()
		}
		if level != currentLevel+1 {
			return fmt.Errorf("reduction level cursor changed")
		}
		var pending int
		if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM v3_reduction_partitions
WHERE run_id = ? AND reduce_key = ? AND level = ? AND status != 'succeeded'`,
			runID, reduceKey, currentLevel).Scan(&pending); err != nil {
			return err
		}
		if pending != 0 {
			return fmt.Errorf("prior reduction level is incomplete")
		}
	}
	ordinalBase := 1_000_000_000 + reduceOrdinal*10_000_000 + level*1_000_000
	for ordinal, input := range partitions {
		nodeKey, err := workflowv3.ReductionPartitionNodeKey(input.Partition, reduced.Policy.FanIn)
		if err != nil {
			return err
		}
		bindingsBody, _ := workflowv3.CanonicalJSON(reduced.Bindings)
		inputSchemas, _ := workflowv3.CanonicalJSON(reduced.InputSchemas)
		outputSchemas, _ := workflowv3.CanonicalJSON(reduced.OutputSchemas)
		modules, _ := workflowv3.CanonicalJSON(reduced.Modules)
		identity := reduced.Implementation
		isolation := workflowv3.EffectivePlanIsolation(reduced.Isolation)
		isolationBody, _ := workflowv3.CanonicalJSON(isolation)
		var budgetAccount, budgetOnExhausted, budgetApprovalGate any
		if reduced.Budget != nil {
			budgetAccount, budgetOnExhausted = reduced.Budget.Account, reduced.Budget.OnExhausted
			if reduced.Budget.ApprovalGate != "" {
				budgetApprovalGate = reduced.Budget.ApprovalGate
			}
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_nodes(
  run_id, node_key, ordinal, task_kind, task_version, bundle_digest,
  entrypoint, task_abi, bindings_json, input_schemas_json,
  output_schemas_json, modules_json, resource_class, max_attempts,
  retry_backoff_ms, budget_account, budget_on_exhausted,
  budget_approval_gate, isolation_class, isolation_policy_digest,
  isolation_executor_digest, isolation_policy_json, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
			runID, nodeKey, ordinalBase+ordinal, identity.Kind, identity.Version,
			identity.BundleDigest, identity.Entrypoint, identity.ABI,
			bindingsBody, inputSchemas, outputSchemas, modules,
			reduced.ResourceClass, reduced.Retry.MaxAttempts, reduced.Retry.BackoffMillis,
			budgetAccount, budgetOnExhausted, budgetApprovalGate,
			isolation.Effective.Class, isolation.PolicyDigest, isolation.ExecutorDigest, isolationBody); err != nil {
			return fmt.Errorf("insert reduction node %d: %w", ordinal, err)
		}
		if err := insertNodeBudget(ctx, tx, runID, nodeKey, reduced.Budget); err != nil {
			return err
		}
		for _, binding := range reduced.Bindings {
			if binding.Source == "gate-output" {
				if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO v3_gate_consumers(run_id, node_key, gate_key)
VALUES (?, ?, ?)`, runID, nodeKey, binding.GateKey); err != nil {
					return err
				}
			}
		}
		for _, dependency := range workflowv3.EffectiveNodeDependencies(reduced.Bindings, nil) {
			if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO v3_dependencies(run_id, node_key, dependency_key)
VALUES (?, ?, ?)`, runID, nodeKey, dependency); err != nil {
				return err
			}
		}
		if err := insertRef(ctx, tx, `
INSERT INTO v3_reduction_partitions(
  run_id, reduce_key, level, ordinal, partition_digest, member_count,
  node_key, input_schema, input_digest, input_media_type,
  input_size_bytes, input_locator, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
			[]any{
				runID, reduceKey, level, ordinal, input.Ref.Digest,
				len(input.Partition.Members), nodeKey,
			}, input.Ref); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_reductions SET source_schema = ?, source_digest = ?,
  source_media_type = ?, source_size_bytes = ?, source_locator = ?,
  source_items = ?, current_level = ?, current_items = ?, status = 'executing',
  updated_at = ?
WHERE run_id = ? AND reduce_key = ? AND status IN ('pending','executing')`,
		source.Schema, source.Digest, source.MediaType, source.Size, source.Locator,
		sourceItems, level, len(partitions), formatTime(now), runID, reduceKey); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, runID, "", "reduction.level_materialized", map[string]any{
		"reduceKey": reduceKey, "level": level, "partitions": len(partitions),
	}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) FailReduction(
	ctx context.Context,
	runID workflowv3.RunID,
	reduceKey string,
	failure workflowv3.Failure,
	now time.Time,
) error {
	if err := workflowv3.ValidateFailure(failure); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `
UPDATE v3_reductions SET status = 'failed', updated_at = ?
WHERE run_id = ? AND reduce_key = ?
  AND status IN ('pending','executing','succeeded')`,
		formatTime(now), runID, reduceKey)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("reduction %s/%s is not fail-able", runID, reduceKey)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_runs SET status = 'failed', updated_at = ?
WHERE run_id = ? AND status = 'running'`, formatTime(now), runID); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, runID, "", "reduction.failed", map[string]any{
		"reduceKey": reduceKey, "class": failure.Class,
		"code": failure.Code, "retryable": failure.Retryable,
	}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) PublishReductionRoot(
	ctx context.Context,
	runID workflowv3.RunID,
	reduceKey string,
	source workflowv3.ArtifactRef,
	sourceItems int,
	root workflowv3.ArtifactRef,
	now time.Time,
) error {
	if err := workflowv3.ValidateArtifactRef(root); err != nil {
		return err
	}
	_, reduced, _, err := s.loadPlanReduction(ctx, runID, reduceKey)
	if err != nil {
		return err
	}
	var expectedSchema string
	for _, schema := range reduced.OutputSchemas {
		expectedSchema = schema
	}
	if root.Schema != expectedSchema {
		return fmt.Errorf("reduction root schema %q does not match %q", root.Schema, expectedSchema)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	var existing workflowv3.ArtifactRef
	var schema, digest, mediaType, locator, existingSourceDigest sql.NullString
	var size sql.NullInt64
	if err := tx.QueryRowContext(ctx, `
SELECT status, root_schema, root_digest, root_media_type, root_size_bytes,
  root_locator, source_digest
FROM v3_reductions WHERE run_id = ? AND reduce_key = ?`,
		runID, reduceKey).Scan(
		&status, &schema, &digest, &mediaType, &size, &locator,
		&existingSourceDigest,
	); err != nil {
		return err
	}
	if existingSourceDigest.Valid && existingSourceDigest.String != source.Digest {
		return fmt.Errorf("reduction source digest changed")
	}
	switch status {
	case "published":
		if schema.Valid && digest.Valid && mediaType.Valid && size.Valid && locator.Valid {
			existing = workflowv3.ArtifactRef{Schema: schema.String, Digest: digest.String, MediaType: mediaType.String, Size: size.Int64, Locator: locator.String}
		}
		if existing == root {
			return tx.Commit()
		}
		return fmt.Errorf("reduction %s/%s already has another root", runID, reduceKey)
	case "pending":
		if sourceItems != 1 {
			return fmt.Errorf("pending reduction can publish only a single source item")
		}
	case "executing":
		var currentItems, succeeded int
		var currentLevel int
		if err := tx.QueryRowContext(ctx, `
SELECT current_items, current_level FROM v3_reductions
WHERE run_id = ? AND reduce_key = ?`, runID, reduceKey).Scan(&currentItems, &currentLevel); err != nil {
			return err
		}
		if currentItems != 1 {
			return fmt.Errorf("reduction current level does not have one root")
		}
		if err := tx.QueryRowContext(ctx, `
SELECT COUNT(*) FROM v3_reduction_partitions partition
JOIN v3_node_outputs output ON output.run_id = partition.run_id AND output.node_key = partition.node_key
WHERE partition.run_id = ? AND partition.reduce_key = ? AND partition.level = ?
  AND partition.status = 'succeeded' AND output.digest = ?`,
			runID, reduceKey, currentLevel, root.Digest).Scan(&succeeded); err != nil {
			return err
		}
		if succeeded != 1 {
			return fmt.Errorf("reduction root does not match the successful partition")
		}
	default:
		return fmt.Errorf("reduction %s/%s is not publishable", runID, reduceKey)
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_reductions SET status = 'published', source_schema = COALESCE(source_schema, ?),
  source_digest = COALESCE(source_digest, ?), source_media_type = COALESCE(source_media_type, ?),
  source_size_bytes = COALESCE(source_size_bytes, ?), source_locator = COALESCE(source_locator, ?),
  source_items = CASE WHEN source_items < 0 THEN ? ELSE source_items END,
  root_schema = ?, root_digest = ?, root_media_type = ?, root_size_bytes = ?,
  root_locator = ?, updated_at = ?
WHERE run_id = ? AND reduce_key = ? AND status IN ('pending','executing')`,
		source.Schema, source.Digest, source.MediaType, source.Size, source.Locator,
		sourceItems, root.Schema, root.Digest, root.MediaType, root.Size, root.Locator,
		formatTime(now), runID, reduceKey); err != nil {
		return err
	}
	if err := insertEvent(ctx, tx, runID, "", "reduction.published", map[string]any{
		"reduceKey": reduceKey, "digest": root.Digest,
	}, now); err != nil {
		return err
	}
	if _, _, err := reconcileRunStateTx(ctx, tx, runID, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) loadPlanReduction(ctx context.Context, runID workflowv3.RunID, reduceKey string) (workflowv3.WorkflowPlan, workflowv3.PlanReduce, int, error) {
	var body []byte
	if err := s.db.QueryRowContext(ctx, `SELECT plan_json FROM v3_runs WHERE run_id = ?`, runID).Scan(&body); err != nil {
		return workflowv3.WorkflowPlan{}, workflowv3.PlanReduce{}, 0, err
	}
	var plan workflowv3.WorkflowPlan
	if err := workflowv3.StrictDecode(body, &plan); err != nil {
		return workflowv3.WorkflowPlan{}, workflowv3.PlanReduce{}, 0, err
	}
	for index, reduced := range plan.Reductions {
		if reduced.Key == reduceKey {
			return plan, reduced, index, nil
		}
	}
	return workflowv3.WorkflowPlan{}, workflowv3.PlanReduce{}, 0, fmt.Errorf("run %s has no reduction %q", runID, reduceKey)
}

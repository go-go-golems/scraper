package workflowv3sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type projectionQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// QueueSnapshot derives bounded scheduler observability from authoritative
// run/node/lease records. It does not persist a second mutable queue state.
func (s *Store) QueueSnapshot(
	ctx context.Context,
	registry workflowv3.RegistryResolver,
	capacities map[string]int,
	now time.Time,
) (workflowv3.QueueSnapshot, error) {
	return queueSnapshot(ctx, s.db, registry, capacities, nil, now)
}

func queueSnapshot(
	ctx context.Context,
	queryer projectionQueryer,
	registry workflowv3.RegistryResolver,
	capacities map[string]int,
	runFilter *workflowv3.RunID,
	now time.Time,
) (workflowv3.QueueSnapshot, error) {
	snapshot := workflowv3.QueueSnapshot{
		ActiveByResource: map[string]int{},
		BlockedByReason:  map[string]int{},
	}
	rows, err := queryer.QueryContext(ctx, `
SELECT run_id, resource_class, COUNT(*) FROM v3_nodes
WHERE status = 'running' GROUP BY run_id, resource_class`)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var runID workflowv3.RunID
		var resource string
		var count int
		if err := rows.Scan(&runID, &resource, &count); err != nil {
			_ = rows.Close()
			return snapshot, err
		}
		if runFilter == nil || runID == *runFilter {
			snapshot.ActiveByResource[resource] += count
		}
	}
	if err := rows.Close(); err != nil {
		return snapshot, err
	}

	rows, err = queryer.QueryContext(ctx, `
SELECT e.run_id, e.map_key, e.status, e.total_items, e.next_index,
  e.materialized_items, e.terminal_items, e.max_materialized_ahead
FROM v3_expansions e JOIN v3_runs r ON r.run_id = e.run_id
WHERE r.status = 'running'
ORDER BY e.run_id, e.map_key`)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var progress workflowv3.MapProgress
		var maxAhead int
		if err := rows.Scan(
			&progress.RunID, &progress.MapKey, &progress.Status,
			&progress.TotalItems, &progress.NextIndex, &progress.MaterializedItems,
			&progress.TerminalItems, &maxAhead,
		); err != nil {
			_ = rows.Close()
			return snapshot, err
		}
		if runFilter != nil && progress.RunID != *runFilter {
			continue
		}
		if progress.TotalItems >= 0 {
			progress.BacklogToMaterialize = progress.TotalItems - progress.NextIndex
		}
		progress.BacklogToExecute = progress.MaterializedItems - progress.TerminalItems
		if (progress.Status == "pending" || progress.Status == "expanding") &&
			progress.BacklogToMaterialize > 0 && progress.BacklogToExecute >= maxAhead {
			snapshot.BlockedByReason["map-backpressure"]++
		}
		snapshot.Maps = append(snapshot.Maps, progress)
	}
	if err := rows.Close(); err != nil {
		return snapshot, err
	}

	rows, err = queryer.QueryContext(ctx, `
SELECT reduction.run_id, reduction.reduce_key, reduction.status,
  reduction.source_items, reduction.current_level,
  COUNT(partition.node_key),
  COALESCE(SUM(CASE WHEN partition.status = 'succeeded' THEN 1 ELSE 0 END), 0),
  reduction.root_digest IS NOT NULL,
  COALESCE(reduction.source_digest, expansion.output_digest) IS NOT NULL
FROM v3_reductions reduction
JOIN v3_runs run ON run.run_id = reduction.run_id
LEFT JOIN v3_reduction_partitions partition
  ON partition.run_id = reduction.run_id
 AND partition.reduce_key = reduction.reduce_key
 AND partition.level = reduction.current_level
LEFT JOIN v3_expansions expansion
  ON expansion.run_id = reduction.run_id
 AND expansion.map_key = reduction.source_map_key
 AND expansion.status = 'published'
WHERE run.status = 'running'
GROUP BY reduction.run_id, reduction.reduce_key
ORDER BY reduction.run_id, reduction.reduce_key`)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var progress workflowv3.ReductionProgress
		var sourceReady bool
		if err := rows.Scan(
			&progress.RunID, &progress.ReduceKey, &progress.Status,
			&progress.SourceItems, &progress.CurrentLevel,
			&progress.PartitionsTotal, &progress.PartitionsSucceeded,
			&progress.RootReady, &sourceReady,
		); err != nil {
			_ = rows.Close()
			return snapshot, err
		}
		if runFilter != nil && progress.RunID != *runFilter {
			continue
		}
		switch {
		case progress.Status == "pending" && !sourceReady:
			snapshot.BlockedByReason["reduction-source"]++
		case progress.Status == "executing" && progress.PartitionsSucceeded < progress.PartitionsTotal:
			snapshot.BlockedByReason["reduction-level"]++
		}
		snapshot.Reductions = append(snapshot.Reductions, progress)
	}
	if err := rows.Close(); err != nil {
		return snapshot, err
	}

	rows, err = queryer.QueryContext(ctx, `
SELECT n.run_id, n.task_kind, n.task_version, n.bundle_digest, n.entrypoint,
  n.task_abi, n.modules_json, n.resource_class, n.max_attempts,
  n.retry_backoff_ms, n.ready_at, n.budget_account,
  n.budget_on_exhausted,
  COALESCE((
    SELECT claim.dimension
    FROM v3_node_budget_claims claim
    JOIN v3_budget_accounts account
      ON account.run_id = claim.run_id
     AND account.account = n.budget_account
     AND account.dimension = claim.dimension
    WHERE claim.run_id = n.run_id AND claim.node_key = n.node_key
      AND account.used_units + account.reserved_units + claim.reserve_units
          > account.limit_units
    ORDER BY claim.dimension LIMIT 1
  ), ''),
  EXISTS (
    SELECT 1 FROM v3_gate_consumers consumer
    JOIN v3_gates gate
      ON gate.run_id = consumer.run_id AND gate.gate_key = consumer.gate_key
    WHERE consumer.run_id = n.run_id AND consumer.node_key = n.node_key
      AND gate.status != 'approved'
  ),
  EXISTS (
    SELECT 1 FROM v3_dependencies d
    JOIN v3_nodes dependency
      ON dependency.run_id = d.run_id
     AND dependency.node_key = d.dependency_key
    WHERE d.run_id = n.run_id AND d.node_key = n.node_key
      AND dependency.status != 'succeeded'
  )
FROM v3_nodes n JOIN v3_runs r ON r.run_id = n.run_id
WHERE n.status = 'pending' AND r.status = 'running'`)
	if err != nil {
		return snapshot, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var runID workflowv3.RunID
		var node workflowv3.PlanNode
		var modules []byte
		var readyAt, budgetAccount, budgetPolicy sql.NullString
		var exhaustedDimension string
		var blockedByGate, blockedByDependency bool
		if err := rows.Scan(
			&runID,
			&node.Implementation.Kind,
			&node.Implementation.Version,
			&node.Implementation.BundleDigest,
			&node.Implementation.Entrypoint,
			&node.Implementation.ABI,
			&modules,
			&node.ResourceClass,
			&node.Retry.MaxAttempts,
			&node.Retry.BackoffMillis,
			&readyAt,
			&budgetAccount,
			&budgetPolicy,
			&exhaustedDimension,
			&blockedByGate,
			&blockedByDependency,
		); err != nil {
			return snapshot, err
		}
		if runFilter != nil && runID != *runFilter {
			continue
		}
		if err := workflowv3.StrictDecode(modules, &node.Modules); err != nil {
			return snapshot, err
		}
		backoffBlocked := false
		if readyAt.Valid {
			deadline, err := time.Parse(time.RFC3339Nano, readyAt.String)
			if err != nil {
				return snapshot, err
			}
			backoffBlocked = deadline.After(now)
		}
		reason := ""
		switch {
		case blockedByGate:
			reason = "gate-dependency"
		case blockedByDependency:
			reason = "dependency"
		case backoffBlocked:
			reason = "retry-backoff"
		case exhaustedDimension != "" && budgetPolicy.String == workflowv3.BudgetExhaustRequireApproval:
			reason = "budget-approval:" + budgetAccount.String + ":" + exhaustedDimension
		case exhaustedDimension != "":
			reason = "budget:" + budgetAccount.String + ":" + exhaustedDimension
		case registry == nil:
			reason = "implementation-unavailable"
		default:
			if _, err := registry.ResolveNode(node); err != nil {
				reason = "implementation-unavailable"
			} else if capacities != nil {
				capacity, ok := capacities[node.ResourceClass]
				if !ok || capacity < 1 || snapshot.ActiveByResource[node.ResourceClass] >= capacity {
					reason = "resource-capacity"
				}
			}
		}
		if reason == "" {
			snapshot.Ready++
		} else {
			snapshot.BlockedByReason[reason]++
		}
	}
	return snapshot, rows.Err()
}

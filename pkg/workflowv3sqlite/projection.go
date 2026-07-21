package workflowv3sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

// QueueSnapshot derives bounded scheduler observability from authoritative
// run/node/lease records. It does not persist a second mutable queue state.
func (s *Store) QueueSnapshot(
	ctx context.Context,
	registry *workflowv3.SealedRegistry,
	capacities map[string]int,
	now time.Time,
) (workflowv3.QueueSnapshot, error) {
	snapshot := workflowv3.QueueSnapshot{
		ActiveByResource: map[string]int{},
		BlockedByReason:  map[string]int{},
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT resource_class, COUNT(*) FROM v3_nodes
WHERE status = 'running' GROUP BY resource_class`)
	if err != nil {
		return snapshot, err
	}
	for rows.Next() {
		var resource string
		var count int
		if err := rows.Scan(&resource, &count); err != nil {
			_ = rows.Close()
			return snapshot, err
		}
		snapshot.ActiveByResource[resource] = count
	}
	if err := rows.Close(); err != nil {
		return snapshot, err
	}

	rows, err = s.db.QueryContext(ctx, `
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

	rows, err = s.db.QueryContext(ctx, `
SELECT n.task_kind, n.task_version, n.bundle_digest, n.entrypoint,
  n.task_abi, n.modules_json, n.resource_class, n.max_attempts,
  n.retry_backoff_ms, n.ready_at,
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
		var node workflowv3.PlanNode
		var modules []byte
		var readyAt sql.NullString
		var blockedByDependency bool
		if err := rows.Scan(
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
			&blockedByDependency,
		); err != nil {
			return snapshot, err
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
		case blockedByDependency:
			reason = "dependency"
		case backoffBlocked:
			reason = "retry-backoff"
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

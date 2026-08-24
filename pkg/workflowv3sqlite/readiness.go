package workflowv3sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

// insertNodeReadiness lowers every binding-owned readiness edge for a durable
// node, regardless of whether the node was declared statically or materialized
// by a map/reduction at runtime.
func insertNodeReadiness(
	ctx context.Context,
	tx *sql.Tx,
	runID workflowv3.RunID,
	nodeKey workflowv3.NodeKey,
	bindings map[string]workflowv3.ValueRef,
	explicit []workflowv3.NodeKey,
) error {
	for _, dependency := range workflowv3.EffectiveNodeDependencies(bindings, explicit) {
		if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO v3_dependencies(run_id, node_key, dependency_key)
VALUES (?, ?, ?)`, runID, nodeKey, dependency); err != nil {
			return fmt.Errorf("insert dependency %s -> %s: %w", nodeKey, dependency, err)
		}
	}
	for _, binding := range bindings {
		switch binding.Source {
		case "gate-output":
			if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO v3_gate_consumers(run_id, node_key, gate_key)
VALUES (?, ?, ?)`, runID, nodeKey, binding.GateKey); err != nil {
				return fmt.Errorf("insert gate consumer %s -> %s: %w", nodeKey, binding.GateKey, err)
			}
		case "reduction-output":
			if _, err := tx.ExecContext(ctx, `
INSERT OR IGNORE INTO v3_reduction_consumers(run_id, node_key, reduce_key)
VALUES (?, ?, ?)`, runID, nodeKey, binding.ReduceKey); err != nil {
				return fmt.Errorf("insert reduction consumer %s -> %s: %w", nodeKey, binding.ReduceKey, err)
			}
		}
	}
	return nil
}

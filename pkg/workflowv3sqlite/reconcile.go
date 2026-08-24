package workflowv3sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

// reconcileRunStateTx derives successful terminal state from durable work and
// output availability. It is idempotent and must run inside the transaction
// that publishes the transition which may complete the run.
func reconcileRunStateTx(
	ctx context.Context,
	tx *sql.Tx,
	runID workflowv3.RunID,
	now time.Time,
) (string, bool, error) {
	var status string
	var planBody []byte
	if err := tx.QueryRowContext(ctx, `
SELECT status, plan_json FROM v3_runs WHERE run_id = ?`, runID).Scan(&status, &planBody); err != nil {
		return "", false, err
	}
	if status != "running" {
		return status, false, nil
	}

	var complete int
	if err := tx.QueryRowContext(ctx, `
SELECT
  NOT EXISTS (SELECT 1 FROM v3_nodes WHERE run_id = ? AND status != 'succeeded')
  AND NOT EXISTS (SELECT 1 FROM v3_expansions WHERE run_id = ? AND status != 'published')
  AND NOT EXISTS (SELECT 1 FROM v3_reductions WHERE run_id = ? AND status != 'published')
  AND NOT EXISTS (
    SELECT 1 FROM v3_gates WHERE run_id = ? AND (
      (budget_activation = 0 AND status != 'approved') OR
      (budget_activation = 1 AND status IN ('waiting','rejected','expired','canceled'))
    )
  )`, runID, runID, runID, runID).Scan(&complete); err != nil {
		return "", false, err
	}
	if complete == 0 {
		return "running", false, nil
	}

	var plan workflowv3.WorkflowPlan
	if err := workflowv3.StrictDecode(planBody, &plan); err != nil {
		return "", false, fmt.Errorf("decode run %s plan for reconciliation: %w", runID, err)
	}
	for _, output := range plan.Outputs {
		if err := scalarOutputReadyTx(ctx, tx, runID, output); err != nil {
			return "", false, err
		}
	}
	for _, output := range plan.SetOutputs {
		if err := setOutputReadyTx(ctx, tx, runID, output); err != nil {
			return "", false, err
		}
	}

	result, err := tx.ExecContext(ctx, `
UPDATE v3_runs SET status = 'succeeded', updated_at = ?
WHERE run_id = ? AND status = 'running'`, formatTime(now), runID)
	if err != nil {
		return "", false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return "", false, err
	}
	if affected == 0 {
		return "running", false, nil
	}
	if err := insertEvent(ctx, tx, runID, "", "run.succeeded", map[string]any{
		"planDigest": plan.Digest,
	}, now); err != nil {
		return "", false, err
	}
	return "succeeded", true, nil
}

func scalarOutputReadyTx(ctx context.Context, tx *sql.Tx, runID workflowv3.RunID, output workflowv3.IROutput) error {
	query := ""
	args := []any{runID}
	switch output.Value.Source {
	case "input":
		query = `SELECT COUNT(*) FROM v3_run_inputs WHERE run_id = ? AND name = ?`
		args = append(args, output.Value.Name)
	case "node-output":
		query = `SELECT COUNT(*) FROM v3_node_outputs WHERE run_id = ? AND node_key = ? AND port = ?`
		args = append(args, output.Value.NodeKey, output.Value.Port)
	case "gate-output":
		query = `SELECT COUNT(*) FROM v3_gates WHERE run_id = ? AND gate_key = ? AND status = 'approved' AND decision_ref_digest IS NOT NULL`
		args = append(args, output.Value.GateKey)
	case "reduction-output":
		query = `SELECT COUNT(*) FROM v3_reductions WHERE run_id = ? AND reduce_key = ? AND status = 'published' AND root_digest IS NOT NULL`
		args = append(args, output.Value.ReduceKey)
	default:
		return fmt.Errorf("run %s output %q has unsupported source %q", runID, output.Name, output.Value.Source)
	}
	return requireResolvedOutput(ctx, tx, runID, output.Name, query, args...)
}

func setOutputReadyTx(ctx context.Context, tx *sql.Tx, runID workflowv3.RunID, output workflowv3.IRSetOutput) error {
	query := ""
	args := []any{runID}
	switch output.Value.Source {
	case "set-input":
		query = `SELECT COUNT(*) FROM v3_run_inputs WHERE run_id = ? AND name = ?`
		args = append(args, output.Value.Name)
	case "map-output":
		query = `SELECT COUNT(*) FROM v3_expansions WHERE run_id = ? AND map_key = ? AND status = 'published' AND output_digest IS NOT NULL`
		args = append(args, output.Value.MapKey)
	default:
		return fmt.Errorf("run %s set output %q has unsupported source %q", runID, output.Name, output.Value.Source)
	}
	return requireResolvedOutput(ctx, tx, runID, output.Name, query, args...)
}

func requireResolvedOutput(
	ctx context.Context,
	tx *sql.Tx,
	runID workflowv3.RunID,
	name string,
	query string,
	args ...any,
) error {
	var count int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("run %s cannot succeed: output %q is unresolved", runID, name)
	}
	return nil
}

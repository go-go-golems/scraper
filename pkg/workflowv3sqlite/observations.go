package workflowv3sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3observations"
)

// ObservationSnapshot reads every authoritative source row through one SQLite
// read transaction. It excludes task payloads, lease capabilities, free-form
// failure messages, event bodies, and artifact locators.
func (s *Store) ObservationSnapshot(ctx context.Context, runID workflowv3.RunID) (workflowv3observations.SourceSnapshot, error) {
	if s == nil || s.db == nil {
		return workflowv3observations.SourceSnapshot{}, fmt.Errorf("workflow v3 store is required")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var source workflowv3observations.SourceSnapshot
	source.Run.RunID = runID
	var planBody []byte
	var created, terminal string
	if err := tx.QueryRowContext(ctx, `
SELECT status,plan_digest,plan_json,created_at,updated_at,
  COALESCE((SELECT MAX(sequence) FROM v3_events WHERE run_id=?),0)
FROM v3_runs WHERE run_id=?`, runID, runID).Scan(
		&source.Run.Status, &source.Run.PlanDigest, &planBody, &created, &terminal, &source.Run.EventSequence,
	); err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	if err := workflowv3.StrictDecode(planBody, &source.Run.Plan); err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	source.Run.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	source.Run.TerminalAt, err = time.Parse(time.RFC3339Nano, terminal)
	if err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}

	rows, err := tx.QueryContext(ctx, `
SELECT n.node_key,n.retry_backoff_ms,n.budget_account,
  EXISTS(SELECT 1 FROM v3_gate_consumers g WHERE g.run_id=n.run_id AND g.node_key=n.node_key),
  EXISTS(SELECT 1 FROM v3_map_items m WHERE m.run_id=n.run_id AND m.node_key=n.node_key),
  EXISTS(SELECT 1 FROM v3_reduction_partitions p WHERE p.run_id=n.run_id AND p.node_key=n.node_key)
FROM v3_nodes n WHERE n.run_id=? ORDER BY n.node_key`, runID)
	if err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	for rows.Next() {
		var node workflowv3observations.NodeSource
		var budget sql.NullString
		var mapped, reduced bool
		if err := rows.Scan(&node.NodeKey, &node.RetryBackoffMillis, &budget, &node.HasGate, &mapped, &reduced); err != nil {
			_ = rows.Close()
			return workflowv3observations.SourceSnapshot{}, err
		}
		node.HasBudget = budget.Valid
		switch {
		case mapped:
			node.Origin = "map-item"
		case reduced:
			node.Origin = "reduction-partition"
		default:
			node.Origin = "static"
		}
		source.Nodes = append(source.Nodes, node)
	}
	if err := rows.Close(); err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	byNode := map[workflowv3.NodeKey]*workflowv3observations.NodeSource{}
	for index := range source.Nodes {
		byNode[source.Nodes[index].NodeKey] = &source.Nodes[index]
	}
	rows, err = tx.QueryContext(ctx, `SELECT node_key,dependency_key FROM v3_dependencies WHERE run_id=? ORDER BY node_key,dependency_key`, runID)
	if err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	for rows.Next() {
		var node, dependency workflowv3.NodeKey
		if err := rows.Scan(&node, &dependency); err != nil {
			_ = rows.Close()
			return workflowv3observations.SourceSnapshot{}, err
		}
		if current := byNode[node]; current != nil {
			current.Dependencies = append(current.Dependencies, dependency)
		}
	}
	if err := rows.Close(); err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}

	rows, err = tx.QueryContext(ctx, `
SELECT node_key,attempt_no,status,registry_generation,resource_class,started_at,finished_at,
  failure_class,failure_code,failure_retryable
FROM v3_attempts WHERE run_id=? ORDER BY node_key,attempt_no`, runID)
	if err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	for rows.Next() {
		var attempt workflowv3observations.AttemptSource
		var started string
		var finished, failureClass, failureCode sql.NullString
		var retryable sql.NullBool
		if err := rows.Scan(&attempt.NodeKey, &attempt.Number, &attempt.Status, &attempt.RegistryGeneration, &attempt.ResourceClass, &started, &finished, &failureClass, &failureCode, &retryable); err != nil {
			_ = rows.Close()
			return workflowv3observations.SourceSnapshot{}, err
		}
		attempt.StartedAt, err = time.Parse(time.RFC3339Nano, started)
		if err != nil {
			_ = rows.Close()
			return workflowv3observations.SourceSnapshot{}, err
		}
		if finished.Valid {
			attempt.FinishedAt, err = time.Parse(time.RFC3339Nano, finished.String)
			if err != nil {
				_ = rows.Close()
				return workflowv3observations.SourceSnapshot{}, err
			}
		}
		if failureClass.Valid {
			attempt.Failure = &workflowv3observations.FailureSource{Class: failureClass.String, Code: failureCode.String, Retryable: retryable.Bool}
		}
		source.Attempts = append(source.Attempts, attempt)
	}
	if err := rows.Close(); err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}

	source.Operations, err = externalOperationsTx(ctx, tx, runID)
	if err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	source.Artifacts, err = observationArtifactsTx(ctx, tx, runID, source.Run.Plan, source.Run.Status)
	if err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	if err := tx.Commit(); err != nil {
		return workflowv3observations.SourceSnapshot{}, err
	}
	return source, nil
}

func observationArtifactsTx(ctx context.Context, tx *sql.Tx, runID workflowv3.RunID, plan workflowv3.WorkflowPlan, status string) ([]workflowv3observations.ArtifactSource, error) {
	var ret []workflowv3observations.ArtifactSource
	appendOutput := func(name string, row *sql.Row) error {
		var artifact workflowv3observations.ArtifactSource
		artifact.Name = name
		if err := row.Scan(&artifact.Schema, &artifact.Digest, &artifact.MediaType, &artifact.SizeBytes); err != nil {
			if err == sql.ErrNoRows && status != "succeeded" {
				return nil
			}
			return fmt.Errorf("resolve observation output %s: %w", name, err)
		}
		ret = append(ret, artifact)
		return nil
	}
	for _, output := range plan.Outputs {
		var row *sql.Row
		switch output.Value.Source {
		case "input":
			row = tx.QueryRowContext(ctx, `SELECT schema_id,digest,media_type,size_bytes FROM v3_run_inputs WHERE run_id=? AND name=?`, runID, output.Value.Name)
		case "node-output":
			row = tx.QueryRowContext(ctx, `SELECT schema_id,digest,media_type,size_bytes FROM v3_node_outputs WHERE run_id=? AND node_key=? AND port=?`, runID, output.Value.NodeKey, output.Value.Port)
		case "gate-output":
			row = tx.QueryRowContext(ctx, `SELECT decision_ref_schema,decision_ref_digest,decision_ref_media_type,decision_ref_size_bytes FROM v3_gates WHERE run_id=? AND gate_key=? AND status='approved'`, runID, output.Value.GateKey)
		case "reduction-output":
			row = tx.QueryRowContext(ctx, `SELECT root_schema,root_digest,root_media_type,root_size_bytes FROM v3_reductions WHERE run_id=? AND reduce_key=? AND status='published'`, runID, output.Value.ReduceKey)
		default:
			return nil, fmt.Errorf("unsupported observation output source %q", output.Value.Source)
		}
		if err := appendOutput(output.Name, row); err != nil {
			return nil, err
		}
	}
	for _, output := range plan.SetOutputs {
		if output.Value.Source != "map-output" {
			return nil, fmt.Errorf("unsupported observation set output source %q", output.Value.Source)
		}
		row := tx.QueryRowContext(ctx, `SELECT output_schema,output_digest,output_media_type,output_size_bytes FROM v3_expansions WHERE run_id=? AND map_key=? AND status='published'`, runID, output.Value.MapKey)
		if err := appendOutput(output.Name, row); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

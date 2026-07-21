package workflowv3sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

var (
	ErrGateVersionConflict = errors.New("gate version conflict")
	ErrGateAlreadyDecided  = errors.New("gate already decided")
	ErrGateExpired         = errors.New("gate expired")
	ErrGateAuthorization   = errors.New("gate decision role denied")
)

func (s *Store) AdvanceOneGate(ctx context.Context, now time.Time) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	var runID workflowv3.RunID
	var gateKey workflowv3.NodeKey
	var timeoutMillis int64
	err = tx.QueryRowContext(ctx, `
SELECT gate.run_id, gate.gate_key, gate.timeout_ms
FROM v3_gates gate JOIN v3_runs run ON run.run_id = gate.run_id
WHERE gate.status = 'pending' AND run.status = 'running'
  AND NOT EXISTS (
    SELECT 1 FROM v3_gate_dependencies dependency
    JOIN v3_nodes node
      ON node.run_id = dependency.run_id
     AND node.node_key = dependency.dependency_key
    WHERE dependency.run_id = gate.run_id
      AND dependency.gate_key = gate.gate_key
      AND node.status != 'succeeded'
  )
  AND (
    gate.budget_activation = 0 OR EXISTS (
      SELECT 1 FROM v3_nodes node
      JOIN v3_node_budget_claims claim
        ON claim.run_id = node.run_id AND claim.node_key = node.node_key
      JOIN v3_budget_accounts account
        ON account.run_id = claim.run_id
       AND account.account = node.budget_account
       AND account.dimension = claim.dimension
      WHERE node.run_id = gate.run_id
        AND node.budget_approval_gate = gate.gate_key
        AND node.status = 'pending'
        AND claim.reserve_units
            > account.limit_units - account.used_units - account.reserved_units
    )
  )
ORDER BY run.created_at, gate.run_id, gate.gate_key LIMIT 1`).Scan(
		&runID, &gateKey, &timeoutMillis,
	)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.Commit(); err != nil {
			return false, err
		}
		return false, nil
	}
	if err != nil {
		return false, err
	}
	stamp := formatTime(now)
	var expires any
	if timeoutMillis > 0 {
		expires = formatTime(now.Add(time.Duration(timeoutMillis) * time.Millisecond))
	}
	result, err := tx.ExecContext(ctx, `
UPDATE v3_gates SET status = 'waiting', version = version + 1,
  requested_at = ?, expires_at = ?
WHERE run_id = ? AND gate_key = ? AND status = 'pending' AND version = 0`,
		stamp, expires, runID, gateKey)
	if err != nil {
		return false, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return false, ErrGateVersionConflict
	}
	if err := insertEvent(ctx, tx, runID, gateKey, "gate.waiting", map[string]any{
		"version": 1, "expiresAt": expires,
	}, now); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) DecideGate(
	ctx context.Context,
	command workflowv3.GateDecisionCommand,
	now time.Time,
) error {
	if err := workflowv3.ValidateGateDecisionCommand(command); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var runStatus, status, decisionSchema, requiredRole string
	var version int64
	var expiresAt, decisionCode, actorID sql.NullString
	var existingSchema, existingDigest, existingMediaType, existingLocator sql.NullString
	var existingSize sql.NullInt64
	if err := tx.QueryRowContext(ctx, `
SELECT run.status, gate.status, gate.version, gate.decision_schema,
  gate.required_role, gate.expires_at, gate.decision_code, gate.actor_id,
  gate.decision_ref_schema, gate.decision_ref_digest,
  gate.decision_ref_media_type, gate.decision_ref_size_bytes,
  gate.decision_ref_locator
FROM v3_gates gate JOIN v3_runs run ON run.run_id = gate.run_id
WHERE gate.run_id = ? AND gate.gate_key = ?`, command.RunID, command.GateKey).Scan(
		&runStatus, &status, &version, &decisionSchema, &requiredRole,
		&expiresAt, &decisionCode, &actorID, &existingSchema, &existingDigest,
		&existingMediaType, &existingSize, &existingLocator,
	); err != nil {
		return err
	}
	if command.AuthorizedRole != requiredRole {
		return ErrGateAuthorization
	}
	targetStatus := "approved"
	if command.Decision == "reject" {
		targetStatus = "rejected"
	}
	if status != "waiting" {
		if status == targetStatus && version == command.ExpectedVersion+1 &&
			decisionCode.String == command.DecisionCode && actorID.String == command.ActorID &&
			decisionRefMatches(existingSchema, existingDigest, existingMediaType, existingSize, existingLocator, command.DecisionRef) {
			return tx.Commit()
		}
		return ErrGateAlreadyDecided
	}
	if runStatus != "running" {
		return ErrGateAlreadyDecided
	}
	if version != command.ExpectedVersion {
		return ErrGateVersionConflict
	}
	if command.DecisionRef != nil && command.DecisionRef.Schema != decisionSchema {
		return fmt.Errorf("gate decision schema mismatch")
	}
	if expiresAt.Valid {
		deadline, err := time.Parse(time.RFC3339Nano, expiresAt.String)
		if err != nil {
			return err
		}
		if !now.Before(deadline) {
			if err := expireGateTx(ctx, tx, command.RunID, command.GateKey, version, now); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return err
			}
			return ErrGateExpired
		}
	}
	var schema, digest, mediaType, locator any
	var size any
	if command.DecisionRef != nil {
		schema, digest = command.DecisionRef.Schema, command.DecisionRef.Digest
		mediaType, size, locator = command.DecisionRef.MediaType, command.DecisionRef.Size, command.DecisionRef.Locator
	}
	stamp := formatTime(now)
	result, err := tx.ExecContext(ctx, `
UPDATE v3_gates SET status = ?, version = version + 1, decided_at = ?,
  decision_code = ?, actor_id = ?, decision_ref_schema = ?,
  decision_ref_digest = ?, decision_ref_media_type = ?,
  decision_ref_size_bytes = ?, decision_ref_locator = ?
WHERE run_id = ? AND gate_key = ? AND status = 'waiting' AND version = ?`,
		targetStatus, stamp, command.DecisionCode, command.ActorID, schema, digest,
		mediaType, size, locator, command.RunID, command.GateKey, version)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrGateVersionConflict
	}
	if err := insertEvent(ctx, tx, command.RunID, command.GateKey, "gate."+targetStatus, map[string]any{
		"version": version + 1, "decisionCode": command.DecisionCode,
		"actorId": command.ActorID, "decisionDigest": digest,
	}, now); err != nil {
		return err
	}
	if targetStatus == "rejected" {
		if err := terminateRunForGateTx(ctx, tx, command.RunID, command.GateKey, "rejected", now); err != nil {
			return err
		}
	} else if err := succeedRunIfCompleteTx(ctx, tx, command.RunID, now); err != nil {
		return err
	}
	return tx.Commit()
}

func decisionRefMatches(
	schema, digest, mediaType sql.NullString,
	size sql.NullInt64,
	locator sql.NullString,
	ref *workflowv3.ArtifactRef,
) bool {
	if ref == nil {
		return !digest.Valid
	}
	return schema.Valid && schema.String == ref.Schema &&
		digest.Valid && digest.String == ref.Digest &&
		mediaType.Valid && mediaType.String == ref.MediaType &&
		size.Valid && size.Int64 == ref.Size &&
		locator.Valid && locator.String == ref.Locator
}

func (s *Store) ExpireDueGates(ctx context.Context, now time.Time) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.QueryContext(ctx, `
SELECT gate.run_id, gate.gate_key, gate.version
FROM v3_gates gate JOIN v3_runs run ON run.run_id = gate.run_id
WHERE gate.status = 'waiting' AND run.status = 'running'
  AND gate.expires_at IS NOT NULL
  AND julianday(gate.expires_at) <= julianday(?)
ORDER BY gate.expires_at, gate.run_id, gate.gate_key LIMIT 64`, formatTime(now))
	if err != nil {
		return 0, err
	}
	type dueGate struct {
		runID   workflowv3.RunID
		key     workflowv3.NodeKey
		version int64
	}
	var due []dueGate
	for rows.Next() {
		var gate dueGate
		if err := rows.Scan(&gate.runID, &gate.key, &gate.version); err != nil {
			_ = rows.Close()
			return 0, err
		}
		due = append(due, gate)
	}
	if err := rows.Close(); err != nil {
		return 0, err
	}
	expired := 0
	for _, gate := range due {
		if err := expireGateTx(ctx, tx, gate.runID, gate.key, gate.version, now); err != nil {
			if errors.Is(err, ErrGateVersionConflict) || errors.Is(err, ErrGateAlreadyDecided) {
				continue
			}
			return 0, err
		}
		expired++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return expired, nil
}

func expireGateTx(
	ctx context.Context,
	tx *sql.Tx,
	runID workflowv3.RunID,
	gateKey workflowv3.NodeKey,
	version int64,
	now time.Time,
) error {
	stamp := formatTime(now)
	result, err := tx.ExecContext(ctx, `
UPDATE v3_gates SET status = 'expired', version = version + 1,
  decided_at = ?, decision_code = 'GATE_EXPIRED'
WHERE run_id = ? AND gate_key = ? AND status = 'waiting' AND version = ?
  AND expires_at IS NOT NULL AND julianday(expires_at) <= julianday(?)`,
		stamp, runID, gateKey, version, stamp)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrGateVersionConflict
	}
	if err := insertEvent(ctx, tx, runID, gateKey, "gate.expired", map[string]any{
		"version": version + 1, "code": "GATE_EXPIRED",
	}, now); err != nil {
		return err
	}
	return terminateRunForGateTx(ctx, tx, runID, gateKey, "expired", now)
}

func terminateRunForGateTx(
	ctx context.Context,
	tx *sql.Tx,
	runID workflowv3.RunID,
	targetGate workflowv3.NodeKey,
	status string,
	now time.Time,
) error {
	stamp := formatTime(now)
	result, err := tx.ExecContext(ctx, `
UPDATE v3_runs SET status = 'failed', cancel_epoch = cancel_epoch + 1,
  updated_at = ? WHERE run_id = ? AND status = 'running'`, stamp, runID)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrGateAlreadyDecided
	}
	if err := settleConservativeWhere(ctx, tx, "run_id = ?", []any{runID}, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_attempts SET status = 'canceled', finished_at = ?
WHERE run_id = ? AND status = 'running'`, stamp, runID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_nodes SET status = 'canceled', lease_token = NULL,
  lease_cancel_epoch = NULL, lease_expires_at = NULL
WHERE run_id = ? AND status IN ('pending','running')`, runID); err != nil {
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
	rows, err := tx.QueryContext(ctx, `
SELECT gate_key, version FROM v3_gates
WHERE run_id = ? AND gate_key != ? AND status IN ('pending','waiting')`, runID, targetGate)
	if err != nil {
		return err
	}
	type canceledGate struct {
		key     workflowv3.NodeKey
		version int64
	}
	var canceled []canceledGate
	for rows.Next() {
		var gate canceledGate
		if err := rows.Scan(&gate.key, &gate.version); err != nil {
			_ = rows.Close()
			return err
		}
		canceled = append(canceled, gate)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
UPDATE v3_gates SET status = 'canceled', version = version + 1, decided_at = ?
WHERE run_id = ? AND gate_key != ? AND status IN ('pending','waiting')`, stamp, runID, targetGate); err != nil {
		return err
	}
	for _, gate := range canceled {
		if err := insertEvent(ctx, tx, runID, gate.key, "gate.canceled", map[string]any{
			"version": gate.version + 1, "cause": "gate." + status,
		}, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) GatePage(
	ctx context.Context,
	runID workflowv3.RunID,
	after workflowv3.NodeKey,
	limit int,
	now time.Time,
) ([]workflowv3.GateProgress, error) {
	if limit < 1 || limit > 1000 {
		return nil, fmt.Errorf("gate page limit must be between 1 and 1000")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT run_id, gate_key, status, version, required_role, requested_at,
  expires_at, decided_at, COALESCE(decision_code, ''),
  decision_ref_digest IS NOT NULL, budget_activation
FROM v3_gates WHERE run_id = ? AND gate_key > ?
ORDER BY gate_key LIMIT ?`, runID, after, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var gates []workflowv3.GateProgress
	for rows.Next() {
		gate, err := scanGateProgress(rows, now)
		if err != nil {
			return nil, err
		}
		gates = append(gates, gate)
	}
	return gates, rows.Err()
}

func scanGateProgress(rows rowScanner, now time.Time) (workflowv3.GateProgress, error) {
	var gate workflowv3.GateProgress
	var requested, expires, decided sql.NullString
	if err := rows.Scan(
		&gate.RunID, &gate.GateKey, &gate.Status, &gate.Version,
		&gate.RequiredRole, &requested, &expires, &decided,
		&gate.DecisionCode, &gate.HasDecisionArtifact, &gate.BudgetActivation,
	); err != nil {
		return gate, err
	}
	if requested.Valid && gate.Status == "waiting" {
		requestedAt, err := time.Parse(time.RFC3339Nano, requested.String)
		if err != nil {
			return gate, err
		}
		if now.After(requestedAt) {
			gate.WaitingAgeMS = now.Sub(requestedAt).Milliseconds()
		}
	}
	if expires.Valid && gate.Status == "waiting" {
		expiresAt, err := time.Parse(time.RFC3339Nano, expires.String)
		if err != nil {
			return gate, err
		}
		remaining := expiresAt.Sub(now).Milliseconds()
		gate.ExpiresInMS = &remaining
	}
	if decided.Valid {
		decidedAt, err := time.Parse(time.RFC3339Nano, decided.String)
		if err != nil {
			return gate, err
		}
		gate.DecidedAt = &decidedAt
	}
	return gate, nil
}

func succeedRunIfCompleteTx(
	ctx context.Context,
	tx *sql.Tx,
	runID workflowv3.RunID,
	now time.Time,
) error {
	_, err := tx.ExecContext(ctx, `
UPDATE v3_runs SET status = 'succeeded', updated_at = ?
WHERE run_id = ? AND status = 'running'
  AND NOT EXISTS (SELECT 1 FROM v3_nodes WHERE run_id = ? AND status != 'succeeded')
  AND NOT EXISTS (SELECT 1 FROM v3_expansions WHERE run_id = ? AND status != 'published')
  AND NOT EXISTS (SELECT 1 FROM v3_reductions WHERE run_id = ? AND status != 'published')
  AND NOT EXISTS (
    SELECT 1 FROM v3_gates WHERE run_id = ? AND (
      (budget_activation = 0 AND status != 'approved') OR
      (budget_activation = 1 AND status IN ('waiting','rejected','expired','canceled'))
    )
  )`, formatTime(now), runID, runID, runID, runID, runID)
	return err
}

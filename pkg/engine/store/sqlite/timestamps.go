package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func epochMicros(t time.Time) int64 {
	return t.UTC().UnixMicro()
}

func timeFromEpochMicros(value int64) time.Time {
	return time.UnixMicro(value).UTC()
}

func nullableEpochMicros(t *time.Time) any {
	if t == nil {
		return nil
	}
	return epochMicros(*t)
}

func migrationBackfill(version int) func(context.Context, *sql.Tx) error {
	if version != 3 {
		return nil
	}
	return backfillSortableTimestampColumns
}

func backfillSortableTimestampColumns(ctx context.Context, tx *sql.Tx) error {
	columns := []struct {
		name      string
		selectSQL string
		updateSQL string
		nullable  bool
	}{
		{"workflows.created_at", "SELECT rowid, created_at FROM workflows", "UPDATE workflows SET created_at_us = ? WHERE rowid = ?", false},
		{"workflows.updated_at", "SELECT rowid, updated_at FROM workflows", "UPDATE workflows SET updated_at_us = ? WHERE rowid = ?", false},
		{"ops.next_attempt_at", "SELECT rowid, next_attempt_at FROM ops", "UPDATE ops SET next_attempt_at_us = ? WHERE rowid = ?", true},
		{"ops.created_at", "SELECT rowid, created_at FROM ops", "UPDATE ops SET created_at_us = ? WHERE rowid = ?", false},
		{"ops.updated_at", "SELECT rowid, updated_at FROM ops", "UPDATE ops SET updated_at_us = ? WHERE rowid = ?", false},
		{"leases.acquired_at", "SELECT rowid, acquired_at FROM leases", "UPDATE leases SET acquired_at_us = ? WHERE rowid = ?", false},
		{"leases.expires_at", "SELECT rowid, expires_at FROM leases", "UPDATE leases SET expires_at_us = ? WHERE rowid = ?", false},
		{"queue_limit_state.last_refill_at", "SELECT rowid, last_refill_at FROM queue_limit_state", "UPDATE queue_limit_state SET last_refill_at_us = ? WHERE rowid = ?", false},
		{"results.completed_at", "SELECT rowid, completed_at FROM results", "UPDATE results SET completed_at_us = ? WHERE rowid = ?", false},
		{"artifacts.created_at", "SELECT rowid, created_at FROM artifacts", "UPDATE artifacts SET created_at_us = ? WHERE rowid = ?", false},
	}

	for _, column := range columns {
		rows, err := tx.QueryContext(ctx, column.selectSQL)
		if err != nil {
			return fmt.Errorf("query %s: %w", column.name, err)
		}

		for rows.Next() {
			var rowID int64
			var value sql.NullString
			if err := rows.Scan(&rowID, &value); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan %s: %w", column.name, err)
			}
			if !value.Valid || value.String == "" {
				if column.nullable {
					continue
				}
				_ = rows.Close()
				return fmt.Errorf("required legacy timestamp %s is empty", column.name)
			}
			parsed, err := time.Parse(time.RFC3339Nano, value.String)
			if err != nil {
				_ = rows.Close()
				return fmt.Errorf("parse %s %q: %w", column.name, value.String, err)
			}
			if _, err := tx.ExecContext(ctx, column.updateSQL, epochMicros(parsed), rowID); err != nil {
				_ = rows.Close()
				return fmt.Errorf("backfill %s: %w", column.name, err)
			}
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("close %s rows: %w", column.name, err)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate %s: %w", column.name, err)
		}
	}
	return nil
}

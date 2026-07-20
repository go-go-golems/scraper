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
		table    string
		legacy   string
		sortable string
		nullable bool
	}{
		{"workflows", "created_at", "created_at_us", false},
		{"workflows", "updated_at", "updated_at_us", false},
		{"ops", "next_attempt_at", "next_attempt_at_us", true},
		{"ops", "created_at", "created_at_us", false},
		{"ops", "updated_at", "updated_at_us", false},
		{"leases", "acquired_at", "acquired_at_us", false},
		{"leases", "expires_at", "expires_at_us", false},
		{"queue_limit_state", "last_refill_at", "last_refill_at_us", false},
		{"results", "completed_at", "completed_at_us", false},
		{"artifacts", "created_at", "created_at_us", false},
	}

	for _, column := range columns {
		query := fmt.Sprintf("SELECT rowid, %s FROM %s", column.legacy, column.table)
		rows, err := tx.QueryContext(ctx, query)
		if err != nil {
			return fmt.Errorf("query %s.%s: %w", column.table, column.legacy, err)
		}

		for rows.Next() {
			var rowID int64
			var value sql.NullString
			if err := rows.Scan(&rowID, &value); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan %s.%s: %w", column.table, column.legacy, err)
			}
			if !value.Valid || value.String == "" {
				if column.nullable {
					continue
				}
				_ = rows.Close()
				return fmt.Errorf("required legacy timestamp %s.%s is empty", column.table, column.legacy)
			}
			parsed, err := time.Parse(time.RFC3339Nano, value.String)
			if err != nil {
				_ = rows.Close()
				return fmt.Errorf("parse %s.%s %q: %w", column.table, column.legacy, value.String, err)
			}
			update := fmt.Sprintf("UPDATE %s SET %s = ? WHERE rowid = ?", column.table, column.sortable)
			if _, err := tx.ExecContext(ctx, update, epochMicros(parsed), rowID); err != nil {
				_ = rows.Close()
				return fmt.Errorf("backfill %s.%s: %w", column.table, column.sortable, err)
			}
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("close %s.%s rows: %w", column.table, column.legacy, err)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate %s.%s: %w", column.table, column.legacy, err)
		}
	}
	return nil
}

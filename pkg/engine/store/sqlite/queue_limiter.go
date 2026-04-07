package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

type queueLimiterState struct {
	Tokens       float64
	LastRefillAt time.Time
}

func countActiveLeasesForQueue(
	ctx context.Context,
	tx *sql.Tx,
	site model.SiteName,
	queue model.QueueKey,
	now time.Time,
) (int, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM leases l
		 JOIN ops active ON active.id = l.op_id
		 WHERE l.expires_at > ?
		   AND active.site = ?
		   AND active.queue_key = ?`,
		now.UTC().Format(time.RFC3339Nano),
		site,
		queue,
	)

	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count active leases for %s/%s: %w", site, queue, err)
	}
	return count, nil
}

func loadQueueLimiterState(
	ctx context.Context,
	tx *sql.Tx,
	site model.SiteName,
	queue model.QueueKey,
) (queueLimiterState, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT tokens, last_refill_at
		 FROM queue_limit_state
		 WHERE site = ? AND queue_key = ?`,
		site,
		queue,
	)

	var tokens float64
	var lastRefillAt string
	if err := row.Scan(&tokens, &lastRefillAt); err != nil {
		if err == sql.ErrNoRows {
			return queueLimiterState{}, nil
		}
		return queueLimiterState{}, fmt.Errorf("load queue limiter state for %s/%s: %w", site, queue, err)
	}

	parsed, err := time.Parse(time.RFC3339Nano, lastRefillAt)
	if err != nil {
		return queueLimiterState{}, fmt.Errorf("parse last_refill_at for %s/%s: %w", site, queue, err)
	}
	return queueLimiterState{
		Tokens:       tokens,
		LastRefillAt: parsed,
	}, nil
}

func refillQueueLimiterState(
	state queueLimiterState,
	policy model.RateLimitPolicy,
	now time.Time,
) queueLimiterState {
	if policy.Burst <= 0 || policy.RatePerSecond <= 0 {
		return state
	}
	if state.LastRefillAt.IsZero() {
		return queueLimiterState{
			Tokens:       float64(policy.Burst),
			LastRefillAt: now.UTC(),
		}
	}

	elapsed := now.UTC().Sub(state.LastRefillAt).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}
	tokens := state.Tokens + elapsed*policy.RatePerSecond
	burst := float64(policy.Burst)
	if tokens > burst {
		tokens = burst
	}
	return queueLimiterState{
		Tokens:       tokens,
		LastRefillAt: now.UTC(),
	}
}

func upsertQueueLimiterState(
	ctx context.Context,
	tx *sql.Tx,
	site model.SiteName,
	queue model.QueueKey,
	state queueLimiterState,
) error {
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO queue_limit_state(site, queue_key, tokens, last_refill_at)
		 VALUES(?, ?, ?, ?)
		 ON CONFLICT(site, queue_key) DO UPDATE SET
		   tokens = excluded.tokens,
		   last_refill_at = excluded.last_refill_at`,
		site,
		queue,
		state.Tokens,
		state.LastRefillAt.UTC().Format(time.RFC3339Nano),
	); err != nil {
		return fmt.Errorf("upsert queue limiter state for %s/%s: %w", site, queue, err)
	}
	return nil
}

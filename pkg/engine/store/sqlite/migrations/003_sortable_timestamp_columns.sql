-- RFC3339Nano suppresses trailing fractional zeros, so SQLite TEXT comparison is
-- not chronological for values such as 2026-07-20T18:00:01Z and
-- 2026-07-20T18:00:01.5Z. The Go migration hook backfills these columns from
-- the legacy text representation. All scheduling comparisons use epoch
-- microseconds after this migration.
ALTER TABLE workflows ADD COLUMN created_at_us INTEGER;
ALTER TABLE workflows ADD COLUMN updated_at_us INTEGER;

ALTER TABLE ops ADD COLUMN next_attempt_at_us INTEGER;
ALTER TABLE ops ADD COLUMN created_at_us INTEGER;
ALTER TABLE ops ADD COLUMN updated_at_us INTEGER;

ALTER TABLE leases ADD COLUMN acquired_at_us INTEGER;
ALTER TABLE leases ADD COLUMN expires_at_us INTEGER;

ALTER TABLE queue_limit_state ADD COLUMN last_refill_at_us INTEGER;
ALTER TABLE results ADD COLUMN completed_at_us INTEGER;
ALTER TABLE artifacts ADD COLUMN created_at_us INTEGER;

CREATE INDEX IF NOT EXISTS idx_ops_ready_at_us ON ops(status, site, queue_key, next_attempt_at_us, created_at_us);
CREATE INDEX IF NOT EXISTS idx_leases_expires_at_us ON leases(expires_at_us);

CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    site TEXT NOT NULL,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    input_json TEXT NOT NULL,
    metadata_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ops (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    parent_id TEXT,
    site TEXT NOT NULL,
    kind TEXT NOT NULL,
    queue_key TEXT NOT NULL,
    dedup_key TEXT NOT NULL,
    input_json TEXT NOT NULL,
    retry_json TEXT NOT NULL,
    metadata_json TEXT NOT NULL,
    status TEXT NOT NULL,
    retry_state_json TEXT NOT NULL,
    next_attempt_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ops_workflow ON ops(workflow_id);
CREATE INDEX IF NOT EXISTS idx_ops_status_queue ON ops(status, queue_key);

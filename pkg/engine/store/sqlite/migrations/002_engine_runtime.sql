CREATE TABLE IF NOT EXISTS op_dependencies (
    workflow_id TEXT NOT NULL,
    op_id TEXT NOT NULL REFERENCES ops(id) ON DELETE CASCADE,
    depends_on_op_id TEXT NOT NULL REFERENCES ops(id) ON DELETE CASCADE,
    required INTEGER NOT NULL,
    PRIMARY KEY (op_id, depends_on_op_id)
);

CREATE INDEX IF NOT EXISTS idx_op_dependencies_workflow ON op_dependencies(workflow_id, op_id);

CREATE TABLE IF NOT EXISTS leases (
    op_id TEXT PRIMARY KEY REFERENCES ops(id) ON DELETE CASCADE,
    worker_id TEXT NOT NULL,
    token TEXT NOT NULL,
    acquired_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_leases_expires_at ON leases(expires_at);

CREATE TABLE IF NOT EXISTS queue_limit_state (
    site TEXT NOT NULL,
    queue_key TEXT NOT NULL,
    tokens REAL NOT NULL,
    last_refill_at TEXT NOT NULL,
    PRIMARY KEY (site, queue_key)
);

CREATE TABLE IF NOT EXISTS results (
    op_id TEXT PRIMARY KEY REFERENCES ops(id) ON DELETE CASCADE,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    data_json TEXT NOT NULL,
    records_json TEXT NOT NULL,
    emitted_json TEXT NOT NULL,
    emitted_ids_json TEXT NOT NULL,
    error_json TEXT,
    completed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    op_id TEXT NOT NULL REFERENCES ops(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    content_type TEXT NOT NULL,
    metadata_json TEXT NOT NULL,
    body BLOB NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_artifacts_op ON artifacts(op_id);

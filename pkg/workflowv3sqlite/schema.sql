PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS v3_runs (
  run_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  plan_digest TEXT NOT NULL,
  plan_json BLOB NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('running', 'succeeded', 'failed', 'canceled')),
  cancel_epoch INTEGER NOT NULL DEFAULT 0,
  dispatch_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS v3_run_inputs (
  run_id TEXT NOT NULL,
  name TEXT NOT NULL,
  schema_id TEXT NOT NULL,
  digest TEXT NOT NULL,
  media_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  locator TEXT NOT NULL,
  PRIMARY KEY (run_id, name),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_nodes (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  task_kind TEXT NOT NULL,
  task_version TEXT NOT NULL,
  bundle_digest TEXT NOT NULL,
  entrypoint TEXT NOT NULL,
  task_abi TEXT NOT NULL,
  bindings_json BLOB NOT NULL,
  input_schemas_json BLOB NOT NULL,
  output_schemas_json BLOB NOT NULL,
  modules_json BLOB NOT NULL,
  resource_class TEXT NOT NULL DEFAULT 'cpu.default',
  max_attempts INTEGER NOT NULL DEFAULT 1,
  retry_backoff_ms INTEGER NOT NULL DEFAULT 0,
  ready_at TEXT,
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'canceled')),
  attempt_count INTEGER NOT NULL DEFAULT 0,
  lease_token TEXT,
  lease_cancel_epoch INTEGER,
  lease_expires_at TEXT,
  PRIMARY KEY (run_id, node_key),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_expansions (
  run_id TEXT NOT NULL,
  map_key TEXT NOT NULL,
  source_schema TEXT,
  source_digest TEXT,
  source_media_type TEXT,
  source_size_bytes INTEGER,
  source_locator TEXT,
  total_items INTEGER NOT NULL DEFAULT -1,
  next_index INTEGER NOT NULL DEFAULT 0,
  materialized_items INTEGER NOT NULL DEFAULT 0,
  terminal_items INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL CHECK (status IN ('pending', 'expanding', 'expanded', 'succeeded', 'failed', 'canceled')),
  updated_at TEXT NOT NULL,
  PRIMARY KEY (run_id, map_key),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_expansion_pages (
  run_id TEXT NOT NULL,
  map_key TEXT NOT NULL,
  page_no INTEGER NOT NULL,
  first_index INTEGER NOT NULL,
  item_count INTEGER NOT NULL,
  page_digest TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (run_id, map_key, page_no),
  FOREIGN KEY (run_id, map_key) REFERENCES v3_expansions(run_id, map_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_map_items (
  run_id TEXT NOT NULL,
  map_key TEXT NOT NULL,
  item_key TEXT NOT NULL,
  item_index INTEGER NOT NULL,
  node_key TEXT NOT NULL,
  input_schema TEXT NOT NULL,
  input_digest TEXT NOT NULL,
  input_media_type TEXT NOT NULL,
  input_size_bytes INTEGER NOT NULL,
  input_locator TEXT NOT NULL,
  PRIMARY KEY (run_id, map_key, item_key),
  UNIQUE (run_id, map_key, item_index),
  UNIQUE (run_id, node_key),
  FOREIGN KEY (run_id, map_key) REFERENCES v3_expansions(run_id, map_key) ON DELETE CASCADE,
  FOREIGN KEY (run_id, node_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_run_resource_dispatch (
  run_id TEXT NOT NULL,
  resource_class TEXT NOT NULL,
  dispatch_count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (run_id, resource_class),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_dependencies (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  dependency_key TEXT NOT NULL,
  PRIMARY KEY (run_id, node_key, dependency_key),
  FOREIGN KEY (run_id, node_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE,
  FOREIGN KEY (run_id, dependency_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_attempts (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  attempt_no INTEGER NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('running', 'succeeded', 'failed', 'lease_lost', 'canceled')),
  lease_token TEXT NOT NULL,
  cancel_epoch INTEGER NOT NULL,
  registry_generation TEXT NOT NULL,
  resource_class TEXT NOT NULL DEFAULT 'cpu.default',
  started_at TEXT NOT NULL,
  finished_at TEXT,
  failure_class TEXT,
  failure_code TEXT,
  failure_retryable INTEGER,
  failure_message TEXT,
  PRIMARY KEY (run_id, node_key, attempt_no),
  FOREIGN KEY (run_id, node_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_node_outputs (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  port TEXT NOT NULL,
  schema_id TEXT NOT NULL,
  digest TEXT NOT NULL,
  media_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  locator TEXT NOT NULL,
  PRIMARY KEY (run_id, node_key, port),
  FOREIGN KEY (run_id, node_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_events (
  sequence INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT NOT NULL,
  node_key TEXT,
  event_type TEXT NOT NULL,
  payload_json BLOB NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS v3_nodes_ready_idx
  ON v3_nodes(status, ordinal, run_id);
CREATE INDEX IF NOT EXISTS v3_attempts_status_idx
  ON v3_attempts(status, run_id, node_key);
CREATE INDEX IF NOT EXISTS v3_events_run_idx
  ON v3_events(run_id, sequence);
CREATE INDEX IF NOT EXISTS v3_expansions_status_idx
  ON v3_expansions(status, updated_at, run_id, map_key);
CREATE INDEX IF NOT EXISTS v3_map_items_node_idx
  ON v3_map_items(run_id, node_key);

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
  failure_count INTEGER NOT NULL DEFAULT 0,
  budget_account TEXT,
  budget_on_exhausted TEXT,
  budget_approval_gate TEXT,
  isolation_class TEXT NOT NULL DEFAULT 'in-process.trusted',
  isolation_policy_digest TEXT NOT NULL DEFAULT '',
  isolation_executor_digest TEXT NOT NULL DEFAULT '',
  isolation_policy_json BLOB,
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
  page_size INTEGER NOT NULL,
  max_items INTEGER NOT NULL,
  max_materialized_ahead INTEGER NOT NULL,
  total_items INTEGER NOT NULL DEFAULT -1,
  next_index INTEGER NOT NULL DEFAULT 0,
  materialized_items INTEGER NOT NULL DEFAULT 0,
  terminal_items INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL CHECK (status IN ('pending', 'expanding', 'expanded', 'succeeded', 'published', 'failed', 'canceled')),
  output_schema TEXT,
  output_digest TEXT,
  output_media_type TEXT,
  output_size_bytes INTEGER,
  output_locator TEXT,
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

CREATE TABLE IF NOT EXISTS v3_reductions (
  run_id TEXT NOT NULL,
  reduce_key TEXT NOT NULL,
  source_kind TEXT NOT NULL,
  source_name TEXT,
  source_map_key TEXT,
  source_schema TEXT,
  source_digest TEXT,
  source_media_type TEXT,
  source_size_bytes INTEGER,
  source_locator TEXT,
  fan_in INTEGER NOT NULL,
  max_levels INTEGER NOT NULL,
  current_level INTEGER NOT NULL DEFAULT -1,
  source_items INTEGER NOT NULL DEFAULT -1,
  current_items INTEGER NOT NULL DEFAULT -1,
  status TEXT NOT NULL CHECK (status IN ('pending', 'executing', 'succeeded', 'published', 'failed', 'canceled')),
  root_schema TEXT,
  root_digest TEXT,
  root_media_type TEXT,
  root_size_bytes INTEGER,
  root_locator TEXT,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (run_id, reduce_key),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_reduction_partitions (
  run_id TEXT NOT NULL,
  reduce_key TEXT NOT NULL,
  level INTEGER NOT NULL,
  ordinal INTEGER NOT NULL,
  partition_digest TEXT NOT NULL,
  member_count INTEGER NOT NULL,
  node_key TEXT NOT NULL,
  input_schema TEXT NOT NULL,
  input_digest TEXT NOT NULL,
  input_media_type TEXT NOT NULL,
  input_size_bytes INTEGER NOT NULL,
  input_locator TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'canceled')),
  PRIMARY KEY (run_id, reduce_key, level, ordinal),
  UNIQUE (run_id, node_key),
  FOREIGN KEY (run_id, reduce_key) REFERENCES v3_reductions(run_id, reduce_key) ON DELETE CASCADE,
  FOREIGN KEY (run_id, node_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_reduction_consumers (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  reduce_key TEXT NOT NULL,
  PRIMARY KEY (run_id, node_key, reduce_key),
  FOREIGN KEY (run_id, node_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE,
  FOREIGN KEY (run_id, reduce_key) REFERENCES v3_reductions(run_id, reduce_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_run_resource_dispatch (
  run_id TEXT NOT NULL,
  resource_class TEXT NOT NULL,
  dispatch_count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (run_id, resource_class),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_gates (
  run_id TEXT NOT NULL,
  gate_key TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN
    ('pending','waiting','approved','rejected','expired','canceled')),
  version INTEGER NOT NULL DEFAULT 0 CHECK (version >= 0),
  policy_digest TEXT NOT NULL,
  decision_schema TEXT NOT NULL,
  required_role TEXT NOT NULL,
  on_reject TEXT NOT NULL,
  on_expire TEXT NOT NULL,
  timeout_ms INTEGER NOT NULL DEFAULT 0 CHECK (timeout_ms >= 0),
  budget_activation INTEGER NOT NULL DEFAULT 0 CHECK (budget_activation IN (0, 1)),
  requested_at TEXT,
  expires_at TEXT,
  decided_at TEXT,
  decision_code TEXT,
  actor_id TEXT,
  decision_ref_schema TEXT,
  decision_ref_digest TEXT,
  decision_ref_media_type TEXT,
  decision_ref_size_bytes INTEGER CHECK (
    decision_ref_size_bytes IS NULL OR decision_ref_size_bytes >= 0
  ),
  decision_ref_locator TEXT,
  PRIMARY KEY (run_id, gate_key),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_gate_dependencies (
  run_id TEXT NOT NULL,
  gate_key TEXT NOT NULL,
  dependency_key TEXT NOT NULL,
  PRIMARY KEY (run_id, gate_key, dependency_key),
  FOREIGN KEY (run_id, gate_key) REFERENCES v3_gates(run_id, gate_key) ON DELETE CASCADE,
  FOREIGN KEY (run_id, dependency_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_gate_consumers (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  gate_key TEXT NOT NULL,
  PRIMARY KEY (run_id, node_key, gate_key),
  FOREIGN KEY (run_id, node_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE,
  FOREIGN KEY (run_id, gate_key) REFERENCES v3_gates(run_id, gate_key) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_v3_gates_status
  ON v3_gates(status, expires_at, run_id, gate_key);

CREATE TABLE IF NOT EXISTS v3_budget_accounts (
  run_id TEXT NOT NULL,
  account TEXT NOT NULL,
  dimension TEXT NOT NULL,
  limit_units INTEGER NOT NULL CHECK (limit_units >= 0),
  used_units INTEGER NOT NULL DEFAULT 0 CHECK (used_units >= 0),
  reserved_units INTEGER NOT NULL DEFAULT 0 CHECK (reserved_units >= 0),
  policy_digest TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
  updated_at TEXT NOT NULL,
  PRIMARY KEY (run_id, account, dimension),
  FOREIGN KEY (run_id) REFERENCES v3_runs(run_id) ON DELETE CASCADE,
  CHECK (used_units + reserved_units <= limit_units)
);

CREATE TABLE IF NOT EXISTS v3_node_budget_claims (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  dimension TEXT NOT NULL,
  reserve_units INTEGER NOT NULL CHECK (reserve_units > 0),
  PRIMARY KEY (run_id, node_key, dimension),
  FOREIGN KEY (run_id, node_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_budget_reservations (
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  attempt_no INTEGER NOT NULL,
  account TEXT NOT NULL,
  dimension TEXT NOT NULL,
  reserved_units INTEGER NOT NULL CHECK (reserved_units > 0),
  settled_units INTEGER CHECK (settled_units >= 0),
  status TEXT NOT NULL CHECK (status IN ('reserved','settled','conservative','released')),
  created_at TEXT NOT NULL,
  settled_at TEXT,
  PRIMARY KEY (run_id, node_key, attempt_no, dimension),
  FOREIGN KEY (run_id, node_key, attempt_no)
    REFERENCES v3_attempts(run_id, node_key, attempt_no) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_v3_budget_reservations_status
  ON v3_budget_reservations(run_id, status);

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
  isolation_class TEXT NOT NULL DEFAULT 'in-process.trusted',
  isolation_policy_digest TEXT NOT NULL DEFAULT '',
  isolation_executor_digest TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  finished_at TEXT,
  failure_class TEXT,
  failure_code TEXT,
  failure_retryable INTEGER,
  failure_message TEXT,
  PRIMARY KEY (run_id, node_key, attempt_no),
  FOREIGN KEY (run_id, node_key) REFERENCES v3_nodes(run_id, node_key) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_external_operations (
  operation_id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  attempt_no INTEGER NOT NULL,
  ordinal INTEGER NOT NULL CHECK (ordinal > 0),
  kind TEXT NOT NULL,
  kind_version TEXT NOT NULL,
  descriptor_digest TEXT NOT NULL,
  authority_digest TEXT NOT NULL,
  correlation_digest TEXT,
  completion_key_digest TEXT NOT NULL,
  admitted_at TEXT NOT NULL,
  UNIQUE (run_id, node_key, attempt_no, ordinal),
  FOREIGN KEY (run_id, node_key, attempt_no)
    REFERENCES v3_attempts(run_id, node_key, attempt_no) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_external_operation_allocations (
  operation_id TEXT NOT NULL,
  dimension TEXT NOT NULL,
  units INTEGER NOT NULL CHECK (units > 0),
  PRIMARY KEY (operation_id, dimension),
  FOREIGN KEY (operation_id)
    REFERENCES v3_external_operations(operation_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_external_operation_measures (
  operation_id TEXT NOT NULL,
  name TEXT NOT NULL,
  units INTEGER NOT NULL CHECK (units >= 0),
  PRIMARY KEY (operation_id, name),
  FOREIGN KEY (operation_id)
    REFERENCES v3_external_operations(operation_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_external_operation_completions (
  operation_id TEXT PRIMARY KEY,
  provider_started_at TEXT NOT NULL,
  elapsed_micros INTEGER NOT NULL CHECK (elapsed_micros >= 0),
  outcome TEXT NOT NULL CHECK (outcome IN
    ('succeeded','failed','canceled','timed-out','unknown')),
  failure_class TEXT,
  failure_code TEXT,
  accounting_mode TEXT NOT NULL CHECK (accounting_mode IN
    ('actual','conservative','none')),
  completed_at TEXT NOT NULL,
  completion_digest TEXT NOT NULL,
  FOREIGN KEY (operation_id)
    REFERENCES v3_external_operations(operation_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS v3_external_operation_counters (
  operation_id TEXT NOT NULL,
  name TEXT NOT NULL,
  units INTEGER NOT NULL CHECK (units >= 0),
  PRIMARY KEY (operation_id, name),
  FOREIGN KEY (operation_id)
    REFERENCES v3_external_operation_completions(operation_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_v3_external_operations_attempt
  ON v3_external_operations(run_id, node_key, attempt_no, ordinal);
CREATE INDEX IF NOT EXISTS idx_v3_external_operations_kind
  ON v3_external_operations(run_id, kind, admitted_at, operation_id);

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
CREATE INDEX IF NOT EXISTS v3_reductions_status_idx
  ON v3_reductions(status, updated_at, run_id, reduce_key);
CREATE INDEX IF NOT EXISTS v3_reduction_partitions_node_idx
  ON v3_reduction_partitions(run_id, node_key);
CREATE INDEX IF NOT EXISTS v3_reduction_consumers_node_idx
  ON v3_reduction_consumers(run_id, node_key);

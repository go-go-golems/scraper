CREATE TABLE IF NOT EXISTS demo_items (
  run_id TEXT NOT NULL,
  item_key TEXT NOT NULL,
  item_index INTEGER NOT NULL,
  base_value INTEGER NOT NULL,
  squared_value INTEGER NOT NULL,
  label TEXT NOT NULL,
  generated_at TEXT NOT NULL,
  PRIMARY KEY (run_id, item_key)
);

CREATE TABLE IF NOT EXISTS demo_runs (
  run_id TEXT PRIMARY KEY,
  workflow_id TEXT NOT NULL,
  item_count INTEGER NOT NULL,
  total_base INTEGER NOT NULL,
  total_squared INTEGER NOT NULL,
  labels_json TEXT NOT NULL,
  artifact_names_json TEXT NOT NULL,
  completed_at TEXT NOT NULL
);

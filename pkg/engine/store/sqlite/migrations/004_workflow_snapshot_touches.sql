-- Keep workflow snapshot cursors current whenever durable operation state changes.
-- NEW.updated_at values are supplied by store transitions, preserving the
-- scheduler's authoritative transition time and sortable epoch-microsecond form.
CREATE TRIGGER IF NOT EXISTS workflows_touch_on_op_update
AFTER UPDATE OF status, updated_at, updated_at_us ON ops
BEGIN
    UPDATE workflows
    SET updated_at = NEW.updated_at,
        updated_at_us = NEW.updated_at_us
    WHERE id = NEW.workflow_id;
END;

CREATE TRIGGER IF NOT EXISTS workflows_touch_on_op_insert
AFTER INSERT ON ops
BEGIN
    UPDATE workflows
    SET updated_at = NEW.created_at,
        updated_at_us = NEW.created_at_us
    WHERE id = NEW.workflow_id;
END;

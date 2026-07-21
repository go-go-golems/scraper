const task = require("workflow/task");
const fs = require("fs:input");
const database = require("db:sync");

exports.synchronizeCustomers = task.implementation(async (ctx) => {
  let configureDenied = false;
  try {
    database.configure("sqlite3", ":memory:");
  } catch (_) {
    configureDenied = true;
  }
  if (!configureDenied) {
    throw task.failure({
      class: "configuration",
      code: "DB_SYNC_CONFIGURE_ALLOWED",
      retryable: false,
      message: "database configure was unexpectedly allowed",
    });
  }

  const text = await fs.readFile(ctx.input().dataset.path, "utf8");
  const input = JSON.parse(text);
  if (!Array.isArray(input.rows) || input.rows.length !== input.expectedCount ||
      input.rows.length > 1000) {
    throw task.failure({
      class: "validation",
      code: "DB_SYNC_CARDINALITY",
      retryable: false,
      message: "database synchronization cardinality is invalid",
    });
  }
  const ids = new Set(input.rows.map((row) => String(row.id)));
  if (ids.size !== input.rows.length) {
    throw task.failure({
      class: "validation",
      code: "DB_SYNC_DUPLICATE_ID",
      retryable: false,
      message: "database synchronization IDs are not unique",
    });
  }

  const identity = ctx.identity();
  const existing = database.query(
    "SELECT operation_key FROM workflow_sync_operations " +
      "WHERE operation_key = ?",
    identity.operationKey,
  );
  let applied = false;
  if (existing.length === 0) {
    const tx = database.begin();
    try {
      for (const row of input.rows) {
        tx.exec(
          "INSERT INTO workflow_sync_customers(id, email) VALUES (?, ?) " +
            "ON CONFLICT(id) DO UPDATE SET email = excluded.email",
          String(row.id),
          String(row.email),
        );
      }
      tx.exec(
        "INSERT INTO workflow_sync_operations(operation_key, row_count) " +
          "VALUES (?, ?)",
        identity.operationKey,
        input.rows.length,
      );
      tx.exec(
        "INSERT INTO workflow_sync_audit(operation_key) VALUES (?)",
        identity.operationKey,
      );
      tx.commit();
      applied = true;
    } catch (error) {
      try {
        tx.rollback();
      } catch (_) {
        // A successful commit closes the transaction before a later failure.
      }
      throw task.failure({
        class: "transport",
        code: "DB_SYNC_TRANSACTION",
        retryable: true,
        message: "database synchronization transaction failed",
      });
    }
  }

  if (input.crashAfterCommit && identity.attempt === 1) {
    throw task.failure({
      class: "transport",
      code: "DB_SYNC_POST_COMMIT",
      retryable: true,
      message: "simulated crash after committed side effect",
    });
  }

  const result = database.query(
    "SELECT row_count FROM workflow_sync_operations " +
      "WHERE operation_key = ?",
    identity.operationKey,
  );
  if (result.length !== 1 || Number(result[0].row_count) !== input.rows.length) {
    throw task.failure({
      class: "validation",
      code: "DB_SYNC_RESULT_CARDINALITY",
      retryable: false,
      message: "database synchronization result is inconsistent",
    });
  }
  const receipt = await ctx.outputs.putJSON("receipt", {
    schema: "database-sync-receipt-ref/v1",
    value: {
      operationKey: identity.operationKey,
      count: input.rows.length,
      applied,
      configureDenied,
    },
  });
  return task.success({receipt});
});

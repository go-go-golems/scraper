const task = require("workflow/task");
const fs = require("fs:input");

globalThis.__workflowV3AttemptLoads =
  (globalThis.__workflowV3AttemptLoads || 0) + 1;

function assertFreshRuntime() {
  if (globalThis.__workflowV3AttemptLoads !== 1) {
    throw new Error("task runtime was reused across attempts");
  }
}

exports.normalizeCustomers = task.implementation(async (ctx) => {
  assertFreshRuntime();
  ctx.checkpoint();
  const text = await fs.readFile(ctx.input().source.path, "utf8");
  const rows = text.trim().split("\n").map((line) => JSON.parse(line));
  const normalized = rows.map((row) => ({
    id: String(row.id).trim(),
    email: String(row.email).trim().toLowerCase(),
  }));
  const dataset = await ctx.outputs.putJSON("dataset", {
    schema: "normalized-customers-ref/v1",
    value: normalized,
  });
  return task.success({dataset});
});

exports.validateDataset = task.implementation(async (ctx) => {
  assertFreshRuntime();
  ctx.checkpoint();
  const text = await fs.readFile(ctx.input().dataset.path, "utf8");
  const rows = JSON.parse(text);
  const ids = new Set(rows.map((row) => row.id));
  if (ids.size !== rows.length) {
    throw task.failure({
      class: "validation",
      code: "CUSTOMER_DUPLICATE_ID",
      retryable: false,
      message: "customer dataset contains duplicate ids",
    });
  }
  const validatedDataset = await ctx.outputs.putJSON(
    "validatedDataset",
    {
      schema: "validated-customers-ref/v1",
      value: {rows, count: rows.length},
    },
  );
  return task.success({validatedDataset});
});

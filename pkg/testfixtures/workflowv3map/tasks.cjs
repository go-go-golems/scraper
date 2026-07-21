const task = require("workflow/task");
const fs = require("fs:input");

exports.normalizeRecord = task.implementation(async (ctx) => {
  const input = ctx.input().record;
  const body = await fs.readFile(input.path, "utf8");
  const record = JSON.parse(body);
  if (!Number.isInteger(record.index) || typeof record.value !== "string") {
    throw task.failure({
      class: "validation",
      code: "MAP_RECORD_INVALID",
      retryable: false,
      message: "map record is invalid",
    });
  }
  const output = await ctx.outputs.putJSON("normalized", {
    schema: "normalized-map-record/v1",
    value: {index: record.index, value: record.value.toUpperCase()},
  });
  return task.success({normalized: output});
});

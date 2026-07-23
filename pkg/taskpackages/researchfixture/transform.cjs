const task = require("workflow/task");
const fs = require("fs:input");
const operation = require("fixture:operation");

exports.transform = task.implementation(async (ctx) => {
  ctx.checkpoint();
  const admitted = operation.invoke();
  if (!admitted.succeeded) {
    throw task.failure({
      class: "transport",
      code: "FIXTURE_OPERATION_TRANSIENT",
      retryable: true,
      message: "fixture operation requested retry",
    });
  }
  const source = JSON.parse(await fs.readFile(ctx.input().source.path, "utf8"));
  const transformed = await ctx.outputs.putJSON("transformed", {
    schema: "fixture-transformed/v1",
    value: {value: String(source.value).toUpperCase()},
  });
  return task.success({transformed});
});

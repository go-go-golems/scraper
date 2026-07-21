const fs = require("fs:input");
const task = require("workflow/task");

exports.prepare = task.implementation(async ctx => {
  const source = JSON.parse(await fs.readFile(ctx.input().source.path, "utf8"));
  const prepared = await ctx.outputs.putJSON("prepared", {
    schema: "gate-prepared/v1",
    value: {recordCount: source.records.length},
  });
  return task.success({prepared});
});

exports.publish = task.implementation(async ctx => {
  const decision = JSON.parse(
    await fs.readFile(ctx.input().decision.path, "utf8"),
  );
  const published = await ctx.outputs.putJSON("published", {
    schema: "gate-published/v1",
    value: {approved: decision.approved === true},
  });
  return task.success({published});
});

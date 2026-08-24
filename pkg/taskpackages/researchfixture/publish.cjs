const task = require("workflow/task");
const fs = require("fs:input");

exports.publish = task.implementation(async (ctx) => {
  ctx.checkpoint();
  const transformed = JSON.parse(
    await fs.readFile(ctx.input().transformed.path, "utf8"),
  );
  const result = await ctx.outputs.putJSON("result", {
    schema: "fixture-result/v1",
    value: {value: transformed.value, published: true},
  });
  return task.success({result});
});

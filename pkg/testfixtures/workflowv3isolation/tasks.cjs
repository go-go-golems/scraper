const fs = require("fs:input");
const task = require("workflow/task");

exports.crashRetry = task.implementation(async ctx => {
  const exec = require("exec:allowlisted");
  const result = exec.run("fixture.echo", [String(ctx.identity().attempt)]);
  const output = await ctx.outputs.putJSON("output", {
    schema: "isolation-output/v1",
    value: {stdout: result.stdout.trim()},
  });
  return task.success({output});
});

exports.tool = task.implementation(async ctx => {
  const exec = require("exec:allowlisted");
  const result = exec.run("fixture.echo", ["hello"]);
  const output = await ctx.outputs.putJSON("output", {
    schema: "isolation-output/v1",
    value: {stdout: result.stdout.trim()},
  });
  return task.success({output});
});

exports.spin = task.implementation(() => {
  while (true) { /* interrupted by the attempt context or parent kill */ }
});

exports.transform = task.implementation(async ctx => {
  const source = JSON.parse(await fs.readFile(ctx.input().source.path, "utf8"));
  const output = await ctx.outputs.putJSON("output", {
    schema: "isolation-output/v1",
    value: {
      count: source.values.length,
      checksum: source.values.join("|").length,
    },
  });
  return task.success({output});
});

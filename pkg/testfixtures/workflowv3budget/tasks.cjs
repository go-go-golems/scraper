const fs = require("fs:input");
const task = require("workflow/task");

exports.invoke = task.implementation(async ctx => {
  const input = ctx.input();
  const request = JSON.parse(await fs.readFile(input.request.path, "utf8"));
  ctx.usage.report("output_tokens", request.outputTokens);
  ctx.usage.report("requests", 1);
  if (request.fail) {
    throw task.failure({
      class: "provider-5xx",
      code: "BUDGET_FIXTURE_PROVIDER_REJECTED",
      retryable: false,
      message: "fixture provider rejected request",
    });
  }
  const response = await ctx.outputs.putJSON("response", {
    schema: "budget-response/v1",
    value: {accepted: true},
  });
  return task.success({response});
});

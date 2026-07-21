const workflow = require("workflow");
const tasks = require("budget-fixture-tasks");

module.exports = workflow.compile(workflow.define("budget-fixture", plan => {
  plan.budget("provider", {
    limits: {output_tokens: 10, requests: 2},
    policyDigest:
      "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  });
  const request = plan.input("request", {schema: "budget-request/v1"});
  const invocation = plan.task("invoke", tasks.invoke({request}), job => {
    job.budget({
      account: "provider",
      reserve: {output_tokens: 5, requests: 1},
      onExhausted: "block",
    });
  });
  plan.output("response", invocation.output("response"));
}));

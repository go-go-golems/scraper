const workflow = require("workflow");
const tasks = require("gate-fixture-tasks");

module.exports = workflow.compile(workflow.define("gate-fixture", plan => {
  const source = plan.input("source", {schema: "gate-source/v1"});
  const prepared = plan.task("prepare", tasks.prepare({source}));
  const decision = plan.gate("review", {
    schema: "gate-decision/v1",
    timeoutMs: 60000,
    requiredRole: "reviewer.primary",
    onReject: "fail-run",
    onExpire: "fail-run",
  }, gate => gate.after(prepared));
  const published = plan.task("publish", tasks.publish({decision}));
  plan.output("published", published.output("published"));
}));

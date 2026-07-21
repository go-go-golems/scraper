const workflow = require("workflow");
const tasks = require("gate-fixture-tasks");

module.exports = workflow.compile(workflow.define("gate-independent", plan => {
  const source = plan.input("source", {schema: "gate-source/v1"});
  const prepared = plan.task("prepare", tasks.prepare({source}));
  plan.output("prepared", prepared.output("prepared"));
}));

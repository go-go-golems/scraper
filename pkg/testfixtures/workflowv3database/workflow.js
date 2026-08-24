const workflow = require("workflow");
const tasks = require("cookbook-database-sync-tasks");

const definition = workflow.define("database-sync", (plan) => {
  const dataset = plan.input("dataset", {
    schema: "database-sync-dataset-ref/v1",
  });
  const sync = plan.task(
    "synchronize",
    tasks.synchronizeCustomers({dataset}),
  );
  plan.output("receipt", sync.output("receipt"));
});

module.exports = workflow.compile(definition);

const workflow = require("workflow");
const tasks = require("research-runner-fixture-tasks");

const definition = workflow.define("research-runner-fixture", (plan) => {
  const source = plan.input("source", {schema: "fixture-source/v1"});
  const transformed = plan.task("transform", tasks.transform({source}));
  const published = plan.task(
    "publish",
    tasks.publish({transformed: transformed.output("transformed")}),
    (job) => job.after(transformed),
  );
  plan.output("result", published.output("result"));
});

module.exports = workflow.compile(definition);

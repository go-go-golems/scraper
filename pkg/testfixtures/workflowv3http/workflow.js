const workflow = require("workflow");
const tasks = require("cookbook-http-snapshot-tasks");

const definition = workflow.define("http-snapshot", (plan) => {
  const articles = plan.input("articles", {
    schema: "http-article-list-ref/v1",
  });
  const snapshot = plan.task(
    "snapshot",
    tasks.snapshotArticles({articles}),
  );
  plan.output("snapshot", snapshot.output("snapshot"));
});

module.exports = workflow.compile(definition);

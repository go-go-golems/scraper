const workflow = require("workflow");
const tasks = require("cookbook-word-count-tasks");

const definition = workflow.define("bounded-word-count", (plan) => {
  const documents = plan.inputSet("documents", {
    itemSchema: "word-document/v1",
    manifestSchema: "scraper-workflow-item-manifest/v1",
    maxItems: 512,
  });
  const counts = plan.map(
    "count-documents",
    documents,
    (document) => tasks.countWords({document}),
    (map) => map.pageSize(16).maxItems(512).maxMaterializedAhead(32),
  );
  const total = plan.reduce(
    "merge-counts",
    counts,
    (partition) => tasks.mergeCounts({partition}),
    (reduce) => reduce.fanIn(8).maxLevels(4),
  );
  plan.output("count", total);
});

module.exports = workflow.compile(definition);

const workflow = require("workflow");
const tasks = require("cookbook-lazy-map-tasks");

const definition = workflow.define("lazy-map-transform", (plan) => {
  const records = plan.inputSet("records", {
    itemSchema: "map-record/v1",
    manifestSchema: "scraper-workflow-item-manifest/v1",
    maxItems: 2000,
  });
  const normalized = plan.map(
    "normalize-records",
    records,
    (record) => tasks.normalizeRecord({record}),
    (map) => map
      .pageSize(64)
      .maxItems(2000)
      .maxMaterializedAhead(128),
  );
  plan.outputSet("records", normalized);
});

module.exports = workflow.compile(definition);

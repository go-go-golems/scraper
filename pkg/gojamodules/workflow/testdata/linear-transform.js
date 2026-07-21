const workflow = require("workflow");
const tasks = require("cookbook-linear-transform-tasks");

const definition = workflow.define("linear-transform", (p) => {
  const source = p.input("source", {
    schema: "customer-jsonl-ref/v1",
  });

  const normalized = p.task(
    "normalize",
    tasks.normalizeCustomers({source}),
  );

  const validated = p.task(
    "validate",
    tasks.validateDataset({
      dataset: normalized.output("dataset"),
    }),
    (job) => job.after(normalized),
  );

  p.output("dataset", validated.output("validatedDataset"));
});

const validation = workflow.validate(definition);
if (!validation.ok) {
  throw new Error(validation.errors.join("\n"));
}

module.exports = workflow.compile(definition);

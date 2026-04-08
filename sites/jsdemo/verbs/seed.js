doc(`Submit the full js-demo workflow starting at seed.js.

This command only submits the initial durable work. Use \`scraper worker run\`
to execute the queued jobs.`);

__verb__("seed", {
  short: "Submit the full js-demo workflow starting at seed.js",
  fields: {
    count: {
      type: "int",
      help: "Number of item ops to emit from seed.js",
      default: 4
    },
    multiplier: {
      type: "int",
      help: "Multiplier used by generated item scripts",
      default: 3
    },
    prefix: {
      type: "string",
      help: "Prefix used when generating demo item labels",
      default: "demo"
    }
  }
});

function seed(ctx) {
  const values = ctx.values || {};
  const runID = String(ctx.workflow.id);
  const count = Math.max(1, Number(values.count || 4));
  const multiplier = Number(values.multiplier || 3);
  const prefix = String(values.prefix || "demo");
  const seedID = runID + ":seed";

  ctx.setWorkflowName("js-demo seed workflow");
  ctx.emit({
    id: seedID,
    kind: "js",
    queue: "site:js-demo:js",
    dedupKey: "js-demo:" + runID,
    metadata: { script: "seed.js" },
    input: {
      runID: runID,
      count: count,
      multiplier: multiplier,
      prefix: prefix
    }
  });
  ctx.setTargetOpID(seedID + ":summary");

  return {
    data: {
      runID: runID,
      submittedEntrypoint: "seed",
      initialOpID: seedID,
      targetOpID: seedID + ":summary"
    }
  };
}

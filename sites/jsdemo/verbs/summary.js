doc(`Submit the summarize.js join stage together with generated item dependencies.`);

__verb__("summary", {
  short: "Submit summarize.js with generated item dependencies",
  fields: {
    count: {
      type: "int",
      help: "Number of item ops to generate before summarize.js runs",
      default: 4
    },
    multiplier: {
      type: "int",
      help: "Multiplier used by the generated item scripts",
      default: 3
    },
    prefix: {
      type: "string",
      help: "Prefix used when generating demo item labels",
      default: "summary"
    }
  }
});

function summary(ctx) {
  const values = ctx.values || {};
  const runID = String(ctx.workflow.id);
  const count = Math.max(1, Number(values.count || 4));
  const multiplier = Number(values.multiplier || 3);
  const prefix = String(values.prefix || "summary");
  const itemOpIDs = [];

  ctx.setWorkflowName("js-demo summary workflow");

  for (let i = 0; i < count; i += 1) {
    const itemID = runID + ":item:" + String(i + 1).padStart(2, "0");
    itemOpIDs.push(itemID);
    ctx.emit({
      id: itemID,
      kind: "js",
      queue: "site:js-demo:js",
      dedupKey: "js-demo:summary-item:" + runID + ":" + String(i + 1).padStart(2, "0"),
      metadata: { script: "build_item.js" },
      input: {
        runID: runID,
        index: i,
        multiplier: multiplier,
        prefix: prefix
      }
    });
  }

  const summaryID = runID + ":summary";
  ctx.emit({
    id: summaryID,
    kind: "js",
    queue: "site:js-demo:js",
    dedupKey: "js-demo:summary:" + runID,
    dependsOn: itemOpIDs.map(function (opID) {
      return { opID: opID, required: true };
    }),
    metadata: { script: "summarize.js" },
    input: {
      runID: runID,
      itemOpIDs: itemOpIDs
    }
  });
  ctx.setTargetOpID(summaryID);

  return {
    data: {
      runID: runID,
      submittedEntrypoint: "summary",
      initialItemCount: itemOpIDs.length,
      targetOpID: summaryID
    }
  };
}

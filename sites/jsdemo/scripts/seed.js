module.exports = function (ctx) {
  const runID = String(ctx.input.runID || ctx.workflow.id);
  const count = Math.max(1, Number(ctx.input.count || 4));
  const multiplier = Number(ctx.input.multiplier || 3);
  const prefix = String(ctx.input.prefix || "demo");

  const itemOpIDs = [];
  for (let i = 0; i < count; i += 1) {
    const itemID = ctx.emit({
      id: ctx.op.id + ":item:" + String(i + 1).padStart(2, "0"),
      kind: "js",
      queue: "site:js-demo:js",
      dedupKey: "js-demo:item:" + runID + ":" + i,
      metadata: { script: "build_item.js" },
      input: {
        runID: runID,
        index: i,
        multiplier: multiplier,
        prefix: prefix
      }
    });
    itemOpIDs.push(itemID);
  }

  const summaryID = ctx.emit({
    id: ctx.op.id + ":summary",
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

  return {
    data: {
      runID: runID,
      itemOpIDs: itemOpIDs,
      summaryOpID: summaryID
    }
  };
};

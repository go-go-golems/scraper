doc(`Submit a single build_item.js op as its own workflow.`);

__verb__("item", {
  short: "Submit one build_item.js op",
  fields: {
    index: {
      type: "int",
      help: "Zero-based item index to generate",
      default: 0
    },
    multiplier: {
      type: "int",
      help: "Multiplier used by the generated item script",
      default: 3
    },
    prefix: {
      type: "string",
      help: "Prefix used when generating demo item labels",
      default: "item"
    }
  }
});

function item(ctx) {
  const values = ctx.values || {};
  const runID = String(ctx.workflow.id);
  const index = Math.max(0, Number(values.index || 0));
  const multiplier = Number(values.multiplier || 3);
  const prefix = String(values.prefix || "item");
  const itemID = runID + ":item:" + String(index + 1).padStart(2, "0");

  ctx.setWorkflowName("js-demo item workflow");
  ctx.emit({
    id: itemID,
    kind: "js",
    queue: "site:js-demo:js",
    dedupKey: "js-demo:item:" + runID + ":" + String(index + 1).padStart(2, "0"),
    metadata: { script: "build_item.js" },
    input: {
      runID: runID,
      index: index,
      multiplier: multiplier,
      prefix: prefix
    }
  });
  ctx.setTargetOpID(itemID);

  return {
    data: {
      runID: runID,
      submittedEntrypoint: "item",
      initialOpID: itemID,
      targetOpID: itemID
    }
  };
}

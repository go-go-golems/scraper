const siteDB = require("site-db");
const demo = require("./lib/demo.js");

function artifactNames(dep) {
  const ret = [];
  if (!dep || !dep.artifacts) {
    return ret;
  }
  for (let i = 0; i < dep.artifacts.length; i += 1) {
    const artifact = dep.artifacts[i];
    if (artifact && artifact.name) {
      ret.push(String(artifact.name));
    }
  }
  return ret;
}

module.exports = async function (ctx) {
  const runID = String(ctx.input.runID || ctx.workflow.id);
  const itemOpIDs = Array.isArray(ctx.input.itemOpIDs) ? ctx.input.itemOpIDs : [];
  const items = [];
  const names = [];

  for (let i = 0; i < itemOpIDs.length; i += 1) {
    const dep = ctx.dep(String(itemOpIDs[i]));
    if (!dep) {
      return {
        error: {
          code: "missing_dependency",
          message: "missing dependency " + itemOpIDs[i],
          retryable: false
        }
      };
    }
    if (dep.error && dep.error.code) {
      return {
        error: {
          code: "item_failed",
          message: "item dependency " + itemOpIDs[i] + " failed",
          retryable: dep.error.retryable === true,
          details: dep.error
        }
      };
    }
    items.push(dep.data || {});
    names.push.apply(names, artifactNames(dep));
  }

  const totals = await Promise.resolve(demo.summarize(items));
  const summary = {
    runID: runID,
    itemCount: totals.itemCount,
    totalBase: totals.totalBase,
    totalSquared: totals.totalSquared,
    labels: totals.labels,
    artifactNames: names
  };

  siteDB.exec(
    "INSERT INTO demo_runs(run_id, workflow_id, item_count, total_base, total_squared, labels_json, artifact_names_json, completed_at) " +
      "VALUES(?, ?, ?, ?, ?, ?, ?, ?) " +
      "ON CONFLICT(run_id) DO UPDATE SET " +
      "workflow_id = excluded.workflow_id, " +
      "item_count = excluded.item_count, " +
      "total_base = excluded.total_base, " +
      "total_squared = excluded.total_squared, " +
      "labels_json = excluded.labels_json, " +
      "artifact_names_json = excluded.artifact_names_json, " +
      "completed_at = excluded.completed_at",
    runID,
    String(ctx.workflow.id),
    summary.itemCount,
    summary.totalBase,
    summary.totalSquared,
    JSON.stringify(summary.labels),
    JSON.stringify(summary.artifactNames),
    ctx.now
  );

  ctx.writeRecord("js-demo.summary", runID, summary);
  ctx.writeArtifact({
    name: "summary-" + runID + ".json",
    kind: "json",
    contentType: "application/json",
    body: JSON.stringify(summary, null, 2)
  });

  return {
    data: summary
  };
};

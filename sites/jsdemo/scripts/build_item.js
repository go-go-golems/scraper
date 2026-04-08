const siteDB = require("site-db");
const demo = require("./lib/demo.js");

module.exports = async function (ctx) {
  const payload = await Promise.resolve(demo.buildItemPayload(ctx.input));

  siteDB.exec(
    "INSERT INTO demo_items(run_id, item_key, item_index, base_value, squared_value, label, generated_at) " +
      "VALUES(?, ?, ?, ?, ?, ?, ?) " +
      "ON CONFLICT(run_id, item_key) DO UPDATE SET " +
      "item_index = excluded.item_index, " +
      "base_value = excluded.base_value, " +
      "squared_value = excluded.squared_value, " +
      "label = excluded.label, " +
      "generated_at = excluded.generated_at",
    payload.runID,
    payload.itemKey,
    payload.index,
    payload.baseValue,
    payload.squaredValue,
    payload.label,
    ctx.now
  );

  ctx.writeRecord("js-demo.item", payload.itemKey, payload);
  ctx.writeArtifact({
    name: payload.itemKey + ".json",
    kind: "json",
    contentType: "application/json",
    body: JSON.stringify(payload, null, 2)
  });

  return {
    data: payload
  };
};

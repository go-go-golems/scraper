function buildItemPayload(input) {
  const index = Number(input.index || 0);
  const multiplier = Number(input.multiplier || 1);
  const prefix = String(input.prefix || "demo");
  const runID = String(input.runID || "demo-run");
  const baseValue = (index + 1) * multiplier;
  const squaredValue = baseValue * baseValue;
  const itemKey = prefix + "-" + String(index + 1).padStart(2, "0");

  return {
    runID: runID,
    itemKey: itemKey,
    index: index,
    baseValue: baseValue,
    squaredValue: squaredValue,
    label: prefix.toUpperCase() + " item " + (index + 1)
  };
}

function summarize(items) {
  const summary = {
    itemCount: items.length,
    totalBase: 0,
    totalSquared: 0,
    labels: []
  };

  for (let i = 0; i < items.length; i += 1) {
    const item = items[i];
    summary.totalBase += Number(item.baseValue || 0);
    summary.totalSquared += Number(item.squaredValue || 0);
    summary.labels.push(String(item.label || ""));
  }

  return summary;
}

module.exports = {
  buildItemPayload: buildItemPayload,
  summarize: summarize
};

"use strict";

exports.run = async function runNormalize(ctx) {
  const input = ctx.input();
  const dataset = await ctx.outputs.putJSON("dataset", {
    schema: "normalized-customer-dataset-ref/v1",
    value: {sourceDigest: input.source.digest, normalized: true},
  });
  return {dataset};
};

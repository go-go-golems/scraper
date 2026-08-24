"use strict";

exports.run = async function runValidate(ctx) {
  const input = ctx.input();
  if (!input.dataset || !input.dataset.digest) {
    throw new Error("dataset reference is required");
  }
  return {
    acceptedDataset: await ctx.outputs.reference("acceptedDataset", input.dataset),
  };
};

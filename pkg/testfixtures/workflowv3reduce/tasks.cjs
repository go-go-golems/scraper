const task = require("workflow/task");
const fs = require("fs:input");

exports.countWords = task.implementation(async (ctx) => {
  const input = ctx.input().document;
  const document = JSON.parse(await fs.readFile(input.path, "utf8"));
  const counts = {};
  for (const word of document.text.split(/\s+/).filter(Boolean)) {
    counts[word] = (counts[word] || 0) + 1;
  }
  const count = await ctx.outputs.putJSON("count", {
    schema: "word-count/v1",
    value: {counts},
  });
  return task.success({count});
});

exports.mergeCounts = task.implementation(async (ctx) => {
  const partition = ctx.input().partition;
  if (!Array.isArray(partition.members) || partition.members.length === 0 ||
      partition.members.length > 8) {
    throw task.failure({
      class: "validation",
      code: "REDUCTION_PARTITION_INVALID",
      retryable: false,
      message: "reduction partition is empty",
    });
  }
  const counts = {};
  for (const member of partition.members) {
    const value = JSON.parse(await fs.readFile(member.path, "utf8"));
    if (Object.hasOwn(value.counts, "__fail__")) {
      throw task.failure({
        class: "validation",
        code: "REDUCTION_SHARD_INVALID",
        retryable: false,
        message: "reduction shard is invalid",
      });
    }
    for (const [word, count] of Object.entries(value.counts)) {
      counts[word] = (counts[word] || 0) + count;
    }
  }
  const count = await ctx.outputs.putJSON("count", {
    schema: "word-count/v1",
    value: {counts},
  });
  return task.success({count});
});

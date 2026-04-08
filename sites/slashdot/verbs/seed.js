doc(`Submit the Slashdot seed workflow starting at seed.js.

This command only submits the initial durable work. Use \`scraper worker run\`
to execute the queued jobs.`);

__verb__("seed", {
  short: "Submit the Slashdot seed workflow",
  fields: {
    "base-url": {
      type: "string",
      help: "Base URL used for the Slashdot frontpage fetch",
      default: "https://slashdot.org/"
    },
    "max-pages": {
      type: "int",
      help: "Maximum number of listing pages to scrape through pagination",
      default: 1
    }
  }
});

function seed(ctx) {
  const values = ctx.values || {};
  const workflowID = String(ctx.workflow.id);
  const baseURL = String(values["base-url"] || "https://slashdot.org/");
  const maxPages = Math.max(1, Number(values["max-pages"] || 1));
  const seedID = workflowID + ":seed";
  const targetOpID = seedID + ":frontpage-extract";

  ctx.setWorkflowName("slashdot seed workflow");
  ctx.emit({
    id: seedID,
    kind: "js",
    queue: "site:slashdot:js",
    dedupKey: "slashdot:seed:" + baseURL,
    metadata: { script: "seed.js" },
    input: {
      baseURL: baseURL,
      maxPages: maxPages
    }
  });
  ctx.setTargetOpID(targetOpID);

  return {
    data: {
      submittedEntrypoint: "seed",
      initialOpID: seedID,
      targetOpID: targetOpID,
      baseURL: baseURL,
      maxPages: maxPages
    }
  };
}

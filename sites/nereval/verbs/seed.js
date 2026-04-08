doc(`Submit a durable NEREVAL scrape workflow.

This command only inserts the initial work into the engine DB. Run
\`scraper worker run\` to execute the queued list and detail jobs.`);

__verb__("seed", {
  short: "Submit a NEREVAL property scrape workflow",
  fields: {
    town: {
      type: "string",
      help: "Town to scrape from data.nereval.com",
      default: "Providence"
    },
    base_url: {
      type: "string",
      help: "Base URL for the NEREVAL site",
      default: "https://data.nereval.com"
    },
    max_pages: {
      type: "int",
      help: "Maximum number of list pages to crawl sequentially",
      default: 2
    }
  }
});

function seed(ctx) {
  const values = ctx.values || {};
  const runID = String(ctx.workflow.id);
  const town = String(values.town || "Providence");
  const baseURL = String(values.base_url || "https://data.nereval.com");
  const maxPages = Math.max(1, Number(values.max_pages || 2));
  const seedID = runID + ":seed";

  ctx.setWorkflowName("nereval scrape workflow");
  ctx.emit({
    id: seedID,
    kind: "js",
    queue: "site:nereval:js",
    dedupKey: "nereval:seed:" + runID,
    metadata: { script: "seed.js" },
    input: {
      runID: runID,
      town: town,
      baseURL: baseURL,
      maxPages: maxPages
    }
  });
  ctx.setTargetOpID(seedID);

  return {
    data: {
      runID: runID,
      submittedEntrypoint: "seed",
      initialOpID: seedID,
      town: town,
      baseURL: baseURL,
      maxPages: maxPages
    }
  };
}

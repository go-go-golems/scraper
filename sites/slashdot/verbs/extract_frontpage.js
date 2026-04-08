doc(`Submit a direct Slashdot frontpage fetch plus extract workflow.

This command bypasses the seed stage and directly submits one \`http/fetch\`
op plus its dependent \`extract_frontpage.js\` op.`);

__verb__("extractFrontpage", {
  command: "extract-frontpage",
  short: "Submit the Slashdot extract_frontpage.js workflow",
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

function extractFrontpage(ctx) {
  const values = ctx.values || {};
  const workflowID = String(ctx.workflow.id);
  const baseURL = String(values["base-url"] || "https://slashdot.org/");
  const maxPages = Math.max(1, Number(values["max-pages"] || 1));
  const fetchID = workflowID + ":frontpage-fetch";
  const extractID = workflowID + ":frontpage-extract";

  ctx.setWorkflowName("slashdot extract workflow");
  ctx.emit({
    id: fetchID,
    kind: "http/fetch",
    queue: "site:slashdot:http",
    dedupKey: "slashdot:frontpage:" + baseURL,
    input: {
      request: {
        method: "GET",
        url: baseURL
      },
      persistBody: true,
      artifactName: "frontpage.html"
    }
  });

  ctx.emit({
    id: extractID,
    kind: "js",
    queue: "site:slashdot:js",
    dedupKey: "slashdot:frontpage-extract:" + baseURL,
    dependsOn: [{ opID: fetchID, required: true }],
    metadata: { script: "extract_frontpage.js" },
    input: {
      baseURL: baseURL,
      fetchedOpID: fetchID,
      pageNumber: 1,
      maxPages: maxPages
    }
  });
  ctx.setTargetOpID(extractID);

  return {
    data: {
      submittedEntrypoint: "extract-frontpage",
      fetchID: fetchID,
      targetOpID: extractID,
      baseURL: baseURL,
      maxPages: maxPages
    }
  };
}

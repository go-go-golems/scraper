const common = require("./lib/common.js");

module.exports = function (ctx) {
  const runID = String(ctx.input.runID || ctx.workflow.id);
  const town = String(ctx.input.town || "Providence");
  const baseURL = String(ctx.input.baseURL || "https://data.nereval.com").replace(/\/+$/, "");
  const maxPages = Math.max(1, Number(ctx.input.maxPages || 1));
  const listURL = baseURL + "/PropertyList.aspx?town=" + encodeURIComponent(town) + "&Search=";

  const fetchID = ctx.emit({
    id: runID + ":list:page:1:fetch",
    kind: "http/fetch",
    queue: "site:nereval:http",
    dedupKey: "nereval:list:" + town + ":page:1",
    input: {
      request: {
        method: "GET",
        url: listURL
      },
      persistBody: true,
      artifactName: "list.html"
    }
  });

  const extractID = ctx.emit({
    id: runID + ":list:page:1:extract",
    kind: "js",
    queue: "site:nereval:js",
    dedupKey: "nereval:list-extract:" + town + ":page:1",
    dependsOn: [{ opID: fetchID, required: true }],
    metadata: { script: "extract_list.js" },
    input: {
      runID: runID,
      town: town,
      baseURL: baseURL,
      pageNumber: 1,
      maxPages: maxPages,
      fetchedOpID: fetchID
    }
  });

  return {
    data: {
      runID: runID,
      town: town,
      baseURL: baseURL,
      maxPages: maxPages,
      initialFetchID: fetchID,
      initialExtractID: extractID,
      firstListURL: common.toAbsolute(baseURL, listURL)
    }
  };
};

module.exports = function (ctx) {
  const baseURL = ctx.input.baseURL || "https://slashdot.org/";
  const fetchID = ctx.emit({
    id: ctx.op.id + ":frontpage-fetch",
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

  const extractID = ctx.emit({
    id: ctx.op.id + ":frontpage-extract",
    kind: "js",
    queue: "site:slashdot:js",
    dedupKey: "slashdot:frontpage-extract:" + baseURL,
    dependsOn: [{ opID: fetchID, required: true }],
    metadata: { script: "extract_frontpage.js" },
    input: {
      baseURL: baseURL,
      fetchedOpID: fetchID
    }
  });

  return {
    data: {
      fetchID: fetchID,
      extractID: extractID
    }
  };
};

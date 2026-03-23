module.exports = function (ctx) {
  const baseURL = ctx.input.baseURL || "https://news.ycombinator.com/";
  const fetchID = ctx.emit({
    id: ctx.op.id + ":frontpage-fetch",
    kind: "http/fetch",
    queue: "site:hackernews:http",
    dedupKey: "hackernews:frontpage:" + baseURL,
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
    queue: "site:hackernews:js",
    dedupKey: "hackernews:frontpage-extract:" + baseURL,
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

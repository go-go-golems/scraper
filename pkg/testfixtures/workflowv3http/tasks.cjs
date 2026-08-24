const task = require("workflow/task");
const fs = require("fs:input");
const {fetch} = require("fetch:public");

exports.snapshotArticles = task.implementation(async (ctx) => {
  ctx.checkpoint();
  const text = await fs.readFile(ctx.input().articles.path, "utf8");
  const request = JSON.parse(text);
  if (!Array.isArray(request.urls) || request.urls.length < 1 ||
      request.urls.length > 8) {
    throw task.failure({
      class: "validation",
      code: "HTTP_SNAPSHOT_CARDINALITY",
      retryable: false,
      message: "article list cardinality is invalid",
    });
  }
  const articles = [];
  for (const url of request.urls) {
    ctx.checkpoint();
    let response;
    try {
      response = await fetch(String(url), {timeout: "2s"});
    } catch (_) {
      throw task.failure({
        class: "transport",
        code: "HTTP_FETCH_TRANSPORT",
        retryable: true,
        message: "HTTP transport failed",
      });
    }
    if (response.status === 429) {
      throw task.failure({
        class: "rate-limit",
        code: "HTTP_FETCH_RATE_LIMIT",
        retryable: true,
        message: "HTTP server rate limited snapshot",
      });
    }
    if (response.status >= 500) {
      throw task.failure({
        class: "provider-5xx",
        code: "HTTP_FETCH_SERVER",
        retryable: true,
        message: "HTTP server failed snapshot",
      });
    }
    if (!response.ok) {
      throw task.failure({
        class: "validation",
        code: "HTTP_FETCH_STATUS",
        retryable: false,
        message: "HTTP status is not successful",
      });
    }
    const body = await response.text();
    articles.push({index: articles.length, status: response.status, body});
  }
  const snapshot = await ctx.outputs.putJSON("snapshot", {
    schema: "http-article-snapshot-ref/v1",
    value: {articles, count: articles.length},
  });
  return task.success({snapshot});
});

const siteDB = require("site-db");
const frontpage = require("./lib/frontpage.js");

function resolveHTML(dep) {
  if (!dep || !dep.artifacts || dep.artifacts.length === 0) {
    return "";
  }

  for (let i = 0; i < dep.artifacts.length; i += 1) {
    const artifact = dep.artifacts[i];
    if (artifact && artifact.name === "frontpage.html" && artifact.bodyText) {
      return artifact.bodyText;
    }
  }

  return dep.artifacts[0] && dep.artifacts[0].bodyText ? dep.artifacts[0].bodyText : "";
}

module.exports = function (ctx) {
  const fetchedOpID = String(ctx.input.fetchedOpID || "");
  const dep = ctx.dep(fetchedOpID);
  if (!dep) {
    return {
      error: {
        code: "missing_dependency",
        message: "missing fetch dependency " + fetchedOpID,
        retryable: false
      }
    };
  }
  if (dep.error && dep.error.code) {
    return {
      error: {
        code: "fetch_failed",
        message: "frontpage fetch did not succeed",
        retryable: dep.error.retryable === true,
        details: dep.error
      }
    };
  }

  const html = resolveHTML(dep);
  if (html === "") {
    return {
      error: {
        code: "missing_body",
        message: "frontpage fetch did not persist an html artifact",
        retryable: false
      }
    };
  }

  const baseURL =
    String(ctx.input.baseURL || "") ||
    String((((dep.data || {}).response || {}).finalURL) || "") ||
    "https://news.ycombinator.com/";
  const pageNumber = Math.max(1, Number(ctx.input.pageNumber || 1));
  const maxPages = Math.max(1, Number(ctx.input.maxPages || 1));

  const stories = frontpage.extractStories(html, baseURL);
  const nextPageURL = frontpage.extractNextPageURL(html, baseURL);
  for (let i = 0; i < stories.length; i += 1) {
    const story = stories[i];
    siteDB.exec(
      "INSERT INTO stories(story_id, rank, title, story_url, site_name, score, author, age_text, comments_url, comments_count, scraped_at) " +
        "VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) " +
        "ON CONFLICT(story_id) DO UPDATE SET " +
        "rank = excluded.rank, " +
        "title = excluded.title, " +
        "story_url = excluded.story_url, " +
        "site_name = excluded.site_name, " +
        "score = excluded.score, " +
        "author = excluded.author, " +
        "age_text = excluded.age_text, " +
        "comments_url = excluded.comments_url, " +
        "comments_count = excluded.comments_count, " +
        "scraped_at = excluded.scraped_at",
      story.storyID,
      story.rank,
      story.title,
      story.storyURL,
      story.siteName,
      story.score,
      story.author,
      story.ageText,
      story.commentsURL,
      story.commentsCount,
      ctx.now
    );
  }

  let nextFetchID = "";
  let nextExtractID = "";
  if (nextPageURL !== "" && pageNumber < maxPages) {
    nextFetchID = ctx.emit({
      id: ctx.op.id + ":page-" + String(pageNumber + 1) + "-fetch",
      kind: "http/fetch",
      queue: "site:hackernews:http",
      dedupKey: "hackernews:frontpage:" + nextPageURL,
      input: {
        request: {
          method: "GET",
          url: nextPageURL
        },
        persistBody: true,
        artifactName: "frontpage.html"
      }
    });

    nextExtractID = ctx.emit({
      id: ctx.op.id + ":page-" + String(pageNumber + 1) + "-extract",
      kind: "js",
      queue: "site:hackernews:js",
      dedupKey: "hackernews:frontpage-extract:" + nextPageURL,
      dependsOn: [{ opID: nextFetchID, required: true }],
      metadata: { script: "extract_frontpage.js" },
      input: {
        baseURL: nextPageURL,
        fetchedOpID: nextFetchID,
        pageNumber: pageNumber + 1,
        maxPages: maxPages
      }
    });
  }

  return {
    data: {
      pageNumber: pageNumber,
      maxPages: maxPages,
      storyCount: stories.length,
      nextPageURL: nextPageURL,
      nextFetchID: nextFetchID,
      nextExtractID: nextExtractID,
      topStoryIDs: stories.slice(0, 5).map(function (story) {
        return story.storyID;
      })
    }
  };
};

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
    "https://slashdot.org/";

  const stories = frontpage.extractStories(html, baseURL);
  for (let i = 0; i < stories.length; i += 1) {
    const story = stories[i];
    siteDB.exec(
      "INSERT INTO stories(story_id, position, title, story_url, source_name, source_url, comments_url, comments_count, author, department, posted_at_text, scraped_at) " +
        "VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) " +
        "ON CONFLICT(story_id) DO UPDATE SET " +
        "position = excluded.position, " +
        "title = excluded.title, " +
        "story_url = excluded.story_url, " +
        "source_name = excluded.source_name, " +
        "source_url = excluded.source_url, " +
        "comments_url = excluded.comments_url, " +
        "comments_count = excluded.comments_count, " +
        "author = excluded.author, " +
        "department = excluded.department, " +
        "posted_at_text = excluded.posted_at_text, " +
        "scraped_at = excluded.scraped_at",
      story.storyID,
      story.position,
      story.title,
      story.storyURL,
      story.sourceName,
      story.sourceURL,
      story.commentsURL,
      story.commentsCount,
      story.author,
      story.department,
      story.postedAtText,
      ctx.now
    );
  }

  return {
    data: {
      storyCount: stories.length,
      topStoryIDs: stories.slice(0, 5).map(function (story) {
        return story.storyID;
      })
    }
  };
};

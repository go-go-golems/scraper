function decodeHTML(value) {
  return String(value || "")
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&#x27;/g, "'")
    .replace(/&#39;/g, "'")
    .replace(/&quot;/g, '"')
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">");
}

function stripTags(value) {
  return String(value || "").replace(/<[^>]+>/g, "");
}

function normalizeWhitespace(value) {
  return decodeHTML(stripTags(value)).replace(/\s+/g, " ").trim();
}

function toAbsolute(baseURL, href) {
  const value = String(href || "").trim();
  if (value === "") {
    return "";
  }
  if (value.indexOf("http://") === 0 || value.indexOf("https://") === 0) {
    return value;
  }
  if (value.indexOf("//") === 0) {
    return "https:" + value;
  }

  const rootMatch = String(baseURL || "").match(/^(https?:\/\/[^/]+)/);
  const root = rootMatch ? rootMatch[1] : "";
  if (value.charAt(0) === "/") {
    return root ? root + value : value;
  }

  const trimmedBase = String(baseURL || "").replace(/\/+$/, "");
  return trimmedBase ? trimmedBase + "/" + value.replace(/^\/+/, "") : value;
}

function parseCommentCount(value) {
  const match = String(value || "").match(/(\d+)/);
  return match ? parseInt(match[1], 10) : 0;
}

function extractStories(html, baseURL) {
  const stories = [];
  const articleRE = /<article\b[^>]*data-fhid="([^"]+)"[^>]*>([\s\S]*?)<\/article>/g;
  let match;
  let position = 0;
  while ((match = articleRE.exec(html)) !== null) {
    const storyID = match[1];
    const body = match[2];
    const titleMatch = body.match(/class="story-title"[\s\S]*?<a[^>]*href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/);
    if (!titleMatch) {
      continue;
    }

    position += 1;
    const sourceMatch = body.match(/class="story-sourcelnk"[^>]*href="([^"]+)"[^>]*>\s*\(([\s\S]*?)\)\s*<\/a>/);
    const commentsMatch = body.match(/class="comment-bubble">[\s\S]*?<a[^>]*href="([^"]+)"[^>]*>([\s\S]*?)<\/a>/);
    const authorMatch = body.match(/Posted by\s*<a[^>]*>([\s\S]*?)<\/a>/);
    const timeMatch = body.match(/<time[^>]*datetime="([^"]+)"/);
    const deptMatch = body.match(/class="dept-text">([\s\S]*?)<\/span>/);

    stories.push({
      storyID: storyID,
      position: position,
      title: normalizeWhitespace(titleMatch[2]),
      storyURL: toAbsolute(baseURL, titleMatch[1]),
      sourceName: sourceMatch ? normalizeWhitespace(sourceMatch[2]) : "",
      sourceURL: sourceMatch ? toAbsolute(baseURL, sourceMatch[1]) : "",
      commentsURL: commentsMatch ? toAbsolute(baseURL, commentsMatch[1]) : "",
      commentsCount: commentsMatch ? parseCommentCount(commentsMatch[2]) : 0,
      author: authorMatch ? normalizeWhitespace(authorMatch[1]) : "",
      department: deptMatch ? normalizeWhitespace(deptMatch[1]) : "",
      postedAtText: timeMatch ? normalizeWhitespace(timeMatch[1]) : ""
    });
  }

  return stories;
}

function extractNextPageURL(html, baseURL) {
  const olderMatch = String(html || "").match(/<a[^>]*class="prevnextbutact"[^>]*href="([^"]+)"[^>]*>\s*Older/i);
  if (!olderMatch) {
    return "";
  }
  return toAbsolute(baseURL, olderMatch[1]);
}

module.exports = {
  extractStories: extractStories,
  extractNextPageURL: extractNextPageURL
};

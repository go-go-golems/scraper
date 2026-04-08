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

function parseCommentCount(text) {
  const cleaned = decodeHTML(stripTags(text)).trim();
  if (cleaned === "" || cleaned === "discuss") {
    return 0;
  }
  const match = cleaned.match(/(\d+)/);
  return match ? parseInt(match[1], 10) : 0;
}

function extractStories(html, baseURL) {
  const stories = [];
  const rowRE = /<tr class="athing submission" id="([^"]+)">([\s\S]*?)<\/tr>\s*<tr[^>]*>\s*<td colspan="2"><\/td>\s*<td class="subtext">([\s\S]*?)<\/td>\s*<\/tr>/g;
  let match;
  while ((match = rowRE.exec(html)) !== null) {
    const storyID = match[1];
    const titleRow = match[2];
    const subtext = match[3];

    const rankMatch = titleRow.match(/class="rank">(\d+)\./);
    const titleMatch = titleRow.match(/<span class="titleline">\s*<a href="([^"]+)">([\s\S]*?)<\/a>/);
    const siteMatch = titleRow.match(/class="sitestr">([^<]+)</);
    const scoreMatch = subtext.match(/class="score"[^>]*>(\d+) points/);
    const authorMatch = subtext.match(/class="hnuser">([^<]+)</);
    const ageMatch = subtext.match(/class="age"[^>]*><a href="([^"]+)">([^<]+)</);

    const itemLinkMatches = [];
    const itemLinkRE = /<a href="(item\?id=\d+)">([\s\S]*?)<\/a>/g;
    let itemLink;
    while ((itemLink = itemLinkRE.exec(subtext)) !== null) {
      itemLinkMatches.push(itemLink);
    }
    const commentsLink = itemLinkMatches.length > 0 ? itemLinkMatches[itemLinkMatches.length - 1] : null;

    if (!rankMatch || !titleMatch) {
      continue;
    }

    stories.push({
      storyID: storyID,
      rank: parseInt(rankMatch[1], 10),
      title: decodeHTML(stripTags(titleMatch[2])).trim(),
      storyURL: toAbsolute(baseURL, titleMatch[1]),
      siteName: siteMatch ? decodeHTML(siteMatch[1]).trim() : "",
      score: scoreMatch ? parseInt(scoreMatch[1], 10) : 0,
      author: authorMatch ? decodeHTML(authorMatch[1]).trim() : "",
      ageText: ageMatch ? decodeHTML(ageMatch[2]).trim() : "",
      commentsURL: commentsLink ? toAbsolute(baseURL, commentsLink[1]) : "",
      commentsCount: commentsLink ? parseCommentCount(commentsLink[2]) : 0
    });
  }

  return stories;
}

function extractNextPageURL(html, baseURL) {
  const moreMatch = String(html || "").match(/<a[^>]*(?:class=['"]morelink['"][^>]*href=['"]([^'"]+)['"]|href=['"]([^'"]+)['"][^>]*class=['"]morelink['"])[^>]*>/i);
  if (!moreMatch) {
    return "";
  }
  return toAbsolute(baseURL, moreMatch[1] || moreMatch[2]);
}

module.exports = {
  extractStories: extractStories,
  extractNextPageURL: extractNextPageURL
};

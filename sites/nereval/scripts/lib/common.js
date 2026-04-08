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
  return decodeHTML(String(value || "").replace(/<[^>]+>/g, " "))
    .replace(/\s+/g, " ")
    .trim();
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

function uniqueStrings(values) {
  const ret = [];
  const seen = {};
  for (let i = 0; i < values.length; i += 1) {
    const value = String(values[i] || "").trim();
    if (value === "" || seen[value]) {
      continue;
    }
    seen[value] = true;
    ret.push(value);
  }
  return ret;
}

module.exports = {
  decodeHTML: decodeHTML,
  stripTags: stripTags,
  toAbsolute: toAbsolute,
  uniqueStrings: uniqueStrings
};

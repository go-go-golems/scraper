const scraperDB = require("scraper-db");
const siteDB = require("site-db");
const common = require("./lib/common.js");
const extract = require("./lib/extract.js");

function resolveHTML(dep) {
  if (!dep || !dep.artifacts || dep.artifacts.length === 0) {
    return "";
  }

  for (let i = 0; i < dep.artifacts.length; i += 1) {
    const artifact = dep.artifacts[i];
    if (artifact && artifact.name === "list.html" && artifact.bodyText) {
      return artifact.bodyText;
    }
  }

  return dep.artifacts[0] && dep.artifacts[0].bodyText ? dep.artifacts[0].bodyText : "";
}

function hasExistingOp(workflowID, opID) {
  const rows = scraperDB.query(
    "SELECT id FROM ops WHERE workflow_id = ? AND id = ? LIMIT 1",
    workflowID,
    opID
  );
  return Array.isArray(rows) && rows.length > 0;
}

module.exports = function (ctx) {
  const fetchedOpID = String(ctx.input.fetchedOpID || "");
  const dep = ctx.dep(fetchedOpID);
  if (!dep) {
    return {
      error: {
        code: "missing_dependency",
        message: "missing list fetch dependency " + fetchedOpID,
        retryable: false
      }
    };
  }
  if (dep.error && dep.error.code) {
    return {
      error: {
        code: "list_fetch_failed",
        message: "list fetch did not succeed",
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
        message: "list fetch did not persist html",
        retryable: false
      }
    };
  }

  const workflowID = String(ctx.workflow.id);
  const town = String(ctx.input.town || "Providence");
  const baseURL = String(ctx.input.baseURL || "https://data.nereval.com").replace(/\/+$/, "");
  const pageNumber = Math.max(1, Number(ctx.input.pageNumber || 1));
  const maxPages = Math.max(1, Number(ctx.input.maxPages || 1));
  const rows = extract.extractListRows(html, baseURL);
  const state = extract.extractListState(html);

  const detailExtractIDs = [];
  const seenAccounts = {};
  for (let i = 0; i < rows.length; i += 1) {
    const row = rows[i];
    siteDB.exec(
      "INSERT INTO properties(account_number, map_lot, location, town, detail_url, scraped_at) " +
        "VALUES(?, ?, ?, ?, ?, ?) " +
        "ON CONFLICT(account_number) DO UPDATE SET " +
        "map_lot = excluded.map_lot, " +
        "location = excluded.location, " +
        "town = excluded.town, " +
        "detail_url = excluded.detail_url, " +
        "scraped_at = excluded.scraped_at",
      row.accountNumber,
      row.mapLot,
      row.location,
      town,
      row.detailURL,
      ctx.now
    );
    if (row.owner !== "") {
      siteDB.exec(
        "INSERT OR IGNORE INTO owners(account_number, owner_name, owner_order) VALUES(?, ?, 1)",
        row.accountNumber,
        row.owner
      );
    }

    if (seenAccounts[row.accountNumber]) {
      continue;
    }
    seenAccounts[row.accountNumber] = true;

    const detailFetchID = workflowID + ":detail:" + row.accountNumber + ":fetch";
    const detailExtractID = workflowID + ":detail:" + row.accountNumber + ":extract";
    if (!hasExistingOp(workflowID, detailFetchID)) {
      ctx.emit({
        id: detailFetchID,
        kind: "http/fetch",
        queue: "site:nereval:http",
        dedupKey: "nereval:detail:" + row.accountNumber,
        input: {
          request: {
            method: "GET",
            url: row.detailURL
          },
          persistBody: true,
          artifactName: "detail.html"
        }
      });
      ctx.emit({
        id: detailExtractID,
        kind: "js",
        queue: "site:nereval:js",
        dedupKey: "nereval:detail-extract:" + row.accountNumber,
        dependsOn: [{ opID: detailFetchID, required: true }],
        metadata: { script: "extract_detail.js" },
        input: {
          town: town,
          accountNumber: row.accountNumber,
          fetchedOpID: detailFetchID
        }
      });
    }
    detailExtractIDs.push(detailExtractID);
  }

  let nextFetchID = "";
  let nextExtractID = "";
  if (state.hasNextPage && state.viewState !== "" && state.eventValidation !== "" && pageNumber < maxPages) {
    nextFetchID = workflowID + ":list:page:" + String(pageNumber + 1) + ":fetch";
    nextExtractID = workflowID + ":list:page:" + String(pageNumber + 1) + ":extract";
    if (!hasExistingOp(workflowID, nextFetchID)) {
      const listURL = baseURL + "/PropertyList.aspx?town=" + encodeURIComponent(town) + "&Search=";
      ctx.emit({
        id: nextFetchID,
        kind: "http/fetch",
        queue: "site:nereval:http",
        dedupKey: "nereval:list:" + town + ":page:" + String(pageNumber + 1),
        input: {
          request: {
            method: "POST",
            url: listURL,
            form: {
              "__VIEWSTATE": "{{ .input.viewState }}",
              "__EVENTVALIDATION": "{{ .input.eventValidation }}",
              "__EVENTTARGET": "ctl00$PropertyList$GridView1",
              "__EVENTARGUMENT": "{{ .input.pageCommand }}"
            }
          },
          persistBody: true,
          artifactName: "list.html",
          viewState: state.viewState,
          eventValidation: state.eventValidation,
          pageCommand: state.pageCommand
        }
      });
      ctx.emit({
        id: nextExtractID,
        kind: "js",
        queue: "site:nereval:js",
        dedupKey: "nereval:list-extract:" + town + ":page:" + String(pageNumber + 1),
        dependsOn: [{ opID: nextFetchID, required: true }],
        metadata: { script: "extract_list.js" },
        input: {
          runID: workflowID,
          town: town,
          baseURL: baseURL,
          pageNumber: pageNumber + 1,
          maxPages: maxPages,
          fetchedOpID: nextFetchID
        }
      });
    }
  }

  return {
    data: {
      pageNumber: pageNumber,
      maxPages: maxPages,
      rowCount: rows.length,
      uniqueAccounts: Object.keys(seenAccounts),
      nextFetchID: nextFetchID,
      nextExtractID: nextExtractID,
      hasNextPage: state.hasNextPage
    }
  };
};

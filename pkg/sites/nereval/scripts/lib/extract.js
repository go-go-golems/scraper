const common = require("./common.js");

function extractTableByID(html, id) {
  const match = String(html || "").match(new RegExp('<table[^>]*id=["\']' + id + '["\'][^>]*>([\\s\\S]*?)<\\/table>', "i"));
  return match ? match[1] : "";
}

function extractRows(tableHTML) {
  return String(tableHTML || "").match(/<tr\b[\s\S]*?<\/tr>/gi) || [];
}

function extractCells(rowHTML) {
  return String(rowHTML || "").match(/<(td|th)\b[\s\S]*?<\/(td|th)>/gi) || [];
}

function hiddenValue(html, id) {
  const regex = new RegExp('<input[^>]*id=["\']' + id + '["\'][^>]*value=["\']([^"\']*)["\']', "i");
  const match = String(html || "").match(regex);
  return match ? common.decodeHTML(match[1]).trim() : "";
}

function extractListState(html) {
  return {
    viewState: hiddenValue(html, "__VIEWSTATE"),
    eventValidation: hiddenValue(html, "__EVENTVALIDATION"),
    hasNextPage: /Page\$Next/.test(String(html || "")),
    pageCommand: /Page\$Next/.test(String(html || "")) ? "Page$Next" : ""
  };
}

function extractListRows(html, baseURL) {
  const table = extractTableByID(html, "PropertyList_GridView1");
  if (table === "") {
    return [];
  }

  const rows = extractRows(table);
  const seen = {};
  const ret = [];
  for (let i = 0; i < rows.length; i += 1) {
    const cells = extractCells(rows[i]);
    if (cells.length < 4) {
      continue;
    }

    const hrefMatch = cells[0].match(/href=["']([^"']+)["']/i);
    const detailURL = hrefMatch ? common.toAbsolute(baseURL, common.decodeHTML(hrefMatch[1])) : "";
    const accountMatch = detailURL.match(/[?&]accountnumber=(\d+)/i);
    const accountNumber = accountMatch ? accountMatch[1] : "";
    const mapLot = common.stripTags(cells[1]);
    const owner = common.stripTags(cells[2]);
    const location = common.stripTags(cells[3]);
    if (accountNumber === "") {
      continue;
    }

    const key = accountNumber + ":" + owner;
    if (seen[key]) {
      continue;
    }
    seen[key] = true;
    ret.push({
      accountNumber: accountNumber,
      mapLot: mapLot,
      owner: owner,
      location: location,
      detailURL: detailURL
    });
  }

  return ret;
}

function extractTablePairsByID(html, id) {
  const table = extractTableByID(html, id);
  const ret = {};
  if (table === "") {
    return ret;
  }

  const rows = extractRows(table);
  for (let i = 0; i < rows.length; i += 1) {
    const cells = extractCells(rows[i]);
    for (let j = 0; j + 1 < cells.length; j += 2) {
      const label = common.stripTags(cells[j]);
      const value = common.stripTags(cells[j + 1]);
      if (label !== "") {
        ret[label] = value;
      }
    }
  }

  return ret;
}

function extractGridRowsByID(html, id) {
  const table = extractTableByID(html, id);
  if (table === "") {
    return [];
  }
  const rows = extractRows(table);
  if (rows.length < 2) {
    return [];
  }

  const headerCells = extractCells(rows[0]);
  const headers = [];
  for (let i = 0; i < headerCells.length; i += 1) {
    headers.push(common.stripTags(headerCells[i]));
  }

  const ret = [];
  for (let i = 1; i < rows.length; i += 1) {
    const cells = extractCells(rows[i]);
    if (cells.length === 0) {
      continue;
    }
    const entry = {};
    for (let j = 0; j < headers.length; j += 1) {
      entry[headers[j]] = common.stripTags(cells[j] || "");
    }
    ret.push(entry);
  }

  return ret;
}

function extractOwners(locationInfo) {
  const owners = [];
  const keys = Object.keys(locationInfo || {});
  for (let i = 0; i < keys.length; i += 1) {
    const key = keys[i];
    if (/^Owner/i.test(key)) {
      owners.push(locationInfo[key]);
    }
  }
  return common.uniqueStrings(owners);
}

function extractMailingAddress(locationInfo) {
  return {
    address1: String((locationInfo || {})["Mailing Address 1"] || "").trim(),
    address2: String((locationInfo || {})["Mailing Address 2"] || "").trim(),
    address3: String((locationInfo || {})["Mailing Address 3"] || "").trim()
  };
}

function extractDetail(html) {
  return {
    parcel: extractTablePairsByID(html, "ParcelID_ParcelID"),
    assessment: extractTablePairsByID(html, "Assessment_Assessment"),
    priorAssessments: extractGridRowsByID(html, "PriorInformation_GridView2"),
    location: extractTablePairsByID(html, "LocationOwner_Location"),
    building: extractTablePairsByID(html, "BuildingInformation_Building"),
    sales: extractGridRowsByID(html, "SaleInformation_Sales"),
    subAreas: extractGridRowsByID(html, "SubArea_SubArea"),
    land: extractTablePairsByID(html, "LandInformation_Land")
  };
}

module.exports = {
  extractListState: extractListState,
  extractListRows: extractListRows,
  extractDetail: extractDetail,
  extractOwners: extractOwners,
  extractMailingAddress: extractMailingAddress
};

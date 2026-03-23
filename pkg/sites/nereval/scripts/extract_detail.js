const siteDB = require("site-db");
const extract = require("./lib/extract.js");

function resolveHTML(dep) {
  if (!dep || !dep.artifacts || dep.artifacts.length === 0) {
    return "";
  }

  for (let i = 0; i < dep.artifacts.length; i += 1) {
    const artifact = dep.artifacts[i];
    if (artifact && artifact.name === "detail.html" && artifact.bodyText) {
      return artifact.bodyText;
    }
  }

  return dep.artifacts[0] && dep.artifacts[0].bodyText ? dep.artifacts[0].bodyText : "";
}

module.exports = function (ctx) {
  const fetchedOpID = String(ctx.input.fetchedOpID || "");
  const accountNumber = String(ctx.input.accountNumber || "");
  const town = String(ctx.input.town || "");
  const dep = ctx.dep(fetchedOpID);
  if (!dep) {
    return {
      error: {
        code: "missing_dependency",
        message: "missing detail fetch dependency " + fetchedOpID,
        retryable: false
      }
    };
  }
  if (dep.error && dep.error.code) {
    return {
      error: {
        code: "detail_fetch_failed",
        message: "detail fetch did not succeed",
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
        message: "detail fetch did not persist html",
        retryable: false
      }
    };
  }

  const detail = extract.extractDetail(html);
  const parcel = detail.parcel || {};
  const assessment = detail.assessment || {};
  const building = detail.building || {};
  const land = detail.land || {};
  const locationInfo = detail.location || {};
  const owners = extract.extractOwners(locationInfo);
  const mailing = extract.extractMailingAddress(locationInfo);

  siteDB.exec(
    "INSERT INTO properties(account_number, map_lot, location, town, detail_url, state_code, card, user_account, scraped_at) " +
      "VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?) " +
      "ON CONFLICT(account_number) DO UPDATE SET " +
      "map_lot = excluded.map_lot, " +
      "location = excluded.location, " +
      "town = excluded.town, " +
      "state_code = excluded.state_code, " +
      "card = excluded.card, " +
      "user_account = excluded.user_account, " +
      "scraped_at = excluded.scraped_at",
    accountNumber,
    parcel["Map/Lot"] || "",
    locationInfo["Location"] || "",
    town,
    "",
    parcel["State Code"] || "",
    parcel["Card"] || "",
    parcel["User Account"] || "",
    ctx.now
  );

  siteDB.exec("DELETE FROM owners WHERE account_number = ?", accountNumber);
  for (let i = 0; i < owners.length; i += 1) {
    siteDB.exec(
      "INSERT INTO owners(account_number, owner_name, owner_order) VALUES(?, ?, ?)",
      accountNumber,
      owners[i],
      i + 1
    );
  }

  siteDB.exec(
    "INSERT INTO mailing_addresses(account_number, address1, address2, address3) VALUES(?, ?, ?, ?) " +
      "ON CONFLICT(account_number) DO UPDATE SET " +
      "address1 = excluded.address1, address2 = excluded.address2, address3 = excluded.address3",
    accountNumber,
    mailing.address1,
    mailing.address2,
    mailing.address3
  );

  siteDB.exec(
    "INSERT INTO assessments(account_number, land_value, building_value, card_total, parcel_total) VALUES(?, ?, ?, ?, ?) " +
      "ON CONFLICT(account_number) DO UPDATE SET " +
      "land_value = excluded.land_value, building_value = excluded.building_value, " +
      "card_total = excluded.card_total, parcel_total = excluded.parcel_total",
    accountNumber,
    assessment["Land"] || "",
    assessment["Building"] || "",
    assessment["Card Total"] || "",
    assessment["Parcel Total"] || ""
  );

  siteDB.exec(
    "INSERT INTO buildings(account_number, design, year_built, heat, fireplaces, rooms, bedrooms, bathrooms, full_bath, above_grade_area) " +
      "VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?) " +
      "ON CONFLICT(account_number) DO UPDATE SET " +
      "design = excluded.design, year_built = excluded.year_built, heat = excluded.heat, " +
      "fireplaces = excluded.fireplaces, rooms = excluded.rooms, bedrooms = excluded.bedrooms, " +
      "bathrooms = excluded.bathrooms, full_bath = excluded.full_bath, above_grade_area = excluded.above_grade_area",
    accountNumber,
    building["Design"] || "",
    building["Year Built"] || "",
    building["Heat"] || "",
    building["Fireplaces"] || "",
    building["Rooms"] || "",
    building["Bedrooms"] || "",
    building["Bathrooms"] || "",
    building["Full Bath"] || "",
    building["Above Grade Living Area"] || ""
  );

  siteDB.exec(
    "INSERT INTO land(account_number, land_area, neighborhood) VALUES(?, ?, ?) " +
      "ON CONFLICT(account_number) DO UPDATE SET land_area = excluded.land_area, neighborhood = excluded.neighborhood",
    accountNumber,
    land["Land Area"] || "",
    land["Neighborhood"] || ""
  );

  siteDB.exec("DELETE FROM prior_assessments WHERE account_number = ?", accountNumber);
  for (let i = 0; i < detail.priorAssessments.length; i += 1) {
    const row = detail.priorAssessments[i];
    siteDB.exec(
      "INSERT INTO prior_assessments(account_number, fiscal_year, land_value, building_value, outbuilding_value, total_value) VALUES(?, ?, ?, ?, ?, ?)",
      accountNumber,
      row["Fiscal Year"] || "",
      row["Land Value"] || "",
      row["Building Value"] || "",
      row["Outbuilding Value"] || "",
      row["Total Value"] || ""
    );
  }

  siteDB.exec("DELETE FROM sales WHERE account_number = ?", accountNumber);
  for (let i = 0; i < detail.sales.length; i += 1) {
    const row = detail.sales[i];
    siteDB.exec(
      "INSERT INTO sales(account_number, sale_date, sale_price, legal_reference, instrument) VALUES(?, ?, ?, ?, ?)",
      accountNumber,
      row["Sale Date"] || "",
      row["Sale Price"] || "",
      row["Legal Reference"] || "",
      row["Instrument"] || ""
    );
  }

  siteDB.exec("DELETE FROM sub_areas WHERE account_number = ?", accountNumber);
  for (let i = 0; i < detail.subAreas.length; i += 1) {
    const row = detail.subAreas[i];
    siteDB.exec(
      "INSERT INTO sub_areas(account_number, sub_area, net_area) VALUES(?, ?, ?)",
      accountNumber,
      row["Sub Area"] || "",
      row["Net Area"] || ""
    );
  }

  return {
    data: {
      accountNumber: accountNumber,
      owners: owners,
      saleCount: detail.sales.length,
      priorAssessmentCount: detail.priorAssessments.length,
      subAreaCount: detail.subAreas.length,
      parcelTotal: assessment["Parcel Total"] || ""
    }
  };
};

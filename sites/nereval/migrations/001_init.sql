CREATE TABLE IF NOT EXISTS properties (
    account_number TEXT PRIMARY KEY,
    map_lot TEXT,
    location TEXT,
    town TEXT NOT NULL,
    detail_url TEXT,
    state_code TEXT,
    card TEXT,
    user_account TEXT,
    scraped_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS owners (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_number TEXT NOT NULL REFERENCES properties(account_number),
    owner_name TEXT NOT NULL,
    owner_order INTEGER NOT NULL DEFAULT 1,
    UNIQUE(account_number, owner_name)
);

CREATE TABLE IF NOT EXISTS assessments (
    account_number TEXT PRIMARY KEY REFERENCES properties(account_number),
    land_value TEXT,
    building_value TEXT,
    card_total TEXT,
    parcel_total TEXT
);

CREATE TABLE IF NOT EXISTS prior_assessments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_number TEXT NOT NULL REFERENCES properties(account_number),
    fiscal_year TEXT NOT NULL,
    land_value TEXT,
    building_value TEXT,
    outbuilding_value TEXT,
    total_value TEXT,
    UNIQUE(account_number, fiscal_year)
);

CREATE TABLE IF NOT EXISTS buildings (
    account_number TEXT PRIMARY KEY REFERENCES properties(account_number),
    design TEXT,
    year_built TEXT,
    heat TEXT,
    fireplaces TEXT,
    rooms TEXT,
    bedrooms TEXT,
    bathrooms TEXT,
    full_bath TEXT,
    above_grade_area TEXT
);

CREATE TABLE IF NOT EXISTS sales (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_number TEXT NOT NULL REFERENCES properties(account_number),
    sale_date TEXT,
    sale_price TEXT,
    legal_reference TEXT,
    instrument TEXT
);

CREATE TABLE IF NOT EXISTS sub_areas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_number TEXT NOT NULL REFERENCES properties(account_number),
    sub_area TEXT,
    net_area TEXT
);

CREATE TABLE IF NOT EXISTS land (
    account_number TEXT PRIMARY KEY REFERENCES properties(account_number),
    land_area TEXT,
    neighborhood TEXT
);

CREATE TABLE IF NOT EXISTS mailing_addresses (
    account_number TEXT PRIMARY KEY REFERENCES properties(account_number),
    address1 TEXT,
    address2 TEXT,
    address3 TEXT
);

CREATE INDEX IF NOT EXISTS idx_properties_town_location ON properties(town, location);
CREATE INDEX IF NOT EXISTS idx_owners_account ON owners(account_number);
CREATE INDEX IF NOT EXISTS idx_sales_account ON sales(account_number);
CREATE INDEX IF NOT EXISTS idx_sub_areas_account ON sub_areas(account_number);

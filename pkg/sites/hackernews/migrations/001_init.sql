CREATE TABLE IF NOT EXISTS stories (
    story_id TEXT PRIMARY KEY,
    rank INTEGER NOT NULL,
    title TEXT NOT NULL,
    story_url TEXT NOT NULL,
    site_name TEXT NOT NULL,
    score INTEGER NOT NULL,
    author TEXT NOT NULL,
    age_text TEXT NOT NULL,
    comments_url TEXT NOT NULL,
    comments_count INTEGER NOT NULL,
    scraped_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_stories_rank ON stories(rank);

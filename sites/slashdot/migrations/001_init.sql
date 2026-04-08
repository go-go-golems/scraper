CREATE TABLE IF NOT EXISTS stories (
    story_id TEXT PRIMARY KEY,
    position INTEGER NOT NULL,
    title TEXT NOT NULL,
    story_url TEXT NOT NULL,
    source_name TEXT,
    source_url TEXT,
    comments_url TEXT,
    comments_count INTEGER NOT NULL DEFAULT 0,
    author TEXT,
    department TEXT,
    posted_at_text TEXT,
    scraped_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_stories_position ON stories(position);

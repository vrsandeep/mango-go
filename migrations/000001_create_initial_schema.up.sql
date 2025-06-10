-- This is the "up" migration file. It contains the SQL to create our
-- initial database tables for series and chapters.

CREATE TABLE IF NOT EXISTS series (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL UNIQUE,
    path TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS chapters (
    id INTEGER PRIMARY KEY,
    series_id INTEGER NOT NULL,
    path TEXT NOT NULL UNIQUE,
    page_count INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
);

CREATE INDEX idx_series_title ON series (title);
CREATE INDEX idx_chapters_path ON chapters (path);

-- SQUASHED INITIAL SCHEMA MIGRATION
-- This file replaces all previous migrations.

PRAGMA foreign_keys = ON;

-- Users table
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('admin', 'user')),
    created_at TIMESTAMP NOT NULL
);

-- Series table
CREATE TABLE series (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL UNIQUE,
    path TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    thumbnail TEXT,
    custom_cover_url TEXT,
    total_chapters INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_series_title ON series (title);

-- Chapters table
CREATE TABLE chapters (
    id INTEGER PRIMARY KEY,
    series_id INTEGER NOT NULL,
    path TEXT NOT NULL UNIQUE,
    page_count INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    thumbnail TEXT,
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
);

CREATE INDEX idx_chapters_path ON chapters (path);

-- Tags table
CREATE TABLE tags (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE INDEX idx_tags_name ON tags (name);

-- Series-Tags join table
CREATE TABLE series_tags (
    series_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (series_id, tag_id),
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Sessions table
CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expiry TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

-- User chapter progress table
CREATE TABLE user_chapter_progress (
    user_id INTEGER NOT NULL,
    chapter_id INTEGER NOT NULL,
    progress_percent INTEGER NOT NULL DEFAULT 0,
    read BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, chapter_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (chapter_id) REFERENCES chapters(id) ON DELETE CASCADE
);

-- User series settings table
CREATE TABLE user_series_settings (
    user_id INTEGER NOT NULL,
    series_id INTEGER NOT NULL,
    sort_by TEXT NOT NULL DEFAULT 'auto',
    sort_dir TEXT NOT NULL DEFAULT 'asc',
    PRIMARY KEY (user_id, series_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
);

-- Download queue table
CREATE TABLE download_queue (
    id INTEGER PRIMARY KEY,
    series_title TEXT NOT NULL,
    chapter_title TEXT NOT NULL,
    chapter_identifier TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    progress INTEGER NOT NULL DEFAULT 0,
    message TEXT,
    created_at TIMESTAMP NOT NULL
);

-- Subscriptions table
CREATE TABLE subscriptions (
    id INTEGER PRIMARY KEY,
    series_title TEXT NOT NULL,
    series_identifier TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    last_checked_at TIMESTAMP
);
CREATE UNIQUE INDEX idx_subscription_provider_series ON subscriptions (series_identifier, provider_id);

-- Triggers for series chapter count
CREATE TRIGGER after_insert_on_chapters
AFTER INSERT ON chapters
BEGIN
    UPDATE series
    SET total_chapters = total_chapters + 1
    WHERE id = NEW.series_id;
END;

CREATE TRIGGER after_delete_on_chapters
AFTER DELETE ON chapters
BEGIN
    UPDATE series
    SET total_chapters = total_chapters - 1
    WHERE id = OLD.series_id;
END;

-- Foreign key check
PRAGMA foreign_key_check;
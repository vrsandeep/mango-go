PRAGMA foreign_keys = ON;

CREATE TABLE folders (
    id INTEGER PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    parent_id INTEGER,
    thumbnail TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (parent_id) REFERENCES folders(id) ON DELETE CASCADE
);
CREATE INDEX idx_folders_path ON folders (path);
CREATE INDEX idx_folders_parent_id ON folders (parent_id);

CREATE TABLE folder_tags (
    folder_id INTEGER NOT NULL,
    tag_id INTEGER NOT NULL,
    PRIMARY KEY (folder_id, tag_id),
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);

-- Add new columns to chapters table
DROP TABLE IF EXISTS chapters;
CREATE TABLE chapters (
    id INTEGER PRIMARY KEY,
    folder_id INTEGER NOT NULL,
    path TEXT NOT NULL UNIQUE,
    page_count INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    thumbnail TEXT,
    content_hash TEXT,
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE
);

-- Add an index for faster lookups by hash
CREATE INDEX idx_chapters_path ON chapters (path);
CREATE UNIQUE INDEX idx_chapters_content_hash ON chapters (content_hash);

-- Remove old series tables
DROP TABLE IF EXISTS series_tags;
DROP TABLE IF EXISTS user_series_settings;
DROP TABLE IF EXISTS series;

-- User folder settings table
CREATE TABLE user_folder_settings (
    user_id INTEGER NOT NULL,
    folder_id INTEGER NOT NULL,
    sort_by TEXT NOT NULL DEFAULT 'auto',
    sort_dir TEXT NOT NULL DEFAULT 'asc',
    PRIMARY KEY (user_id, folder_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE
);

-- Foreign key check
PRAGMA foreign_key_check;

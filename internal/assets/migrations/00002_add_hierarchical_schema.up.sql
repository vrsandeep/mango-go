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
ALTER TABLE chapters ADD COLUMN folder_id INTEGER;
ALTER TABLE chapters ADD COLUMN content_hash TEXT;
-- Add an index for faster lookups by hash
CREATE UNIQUE INDEX idx_chapters_content_hash ON chapters (content_hash);


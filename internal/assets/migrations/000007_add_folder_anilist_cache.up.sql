PRAGMA foreign_keys = ON;

CREATE TABLE folder_anilist_cache (
    folder_id INTEGER PRIMARY KEY,
    anilist_id INTEGER NOT NULL,
    site_url TEXT NOT NULL,
    cover_image_url TEXT,
    title_romaji TEXT,
    title_english TEXT,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (folder_id) REFERENCES folders(id) ON DELETE CASCADE
);

-- Foreign key check
PRAGMA foreign_key_check;

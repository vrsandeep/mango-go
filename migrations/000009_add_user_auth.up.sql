-- Create all tables for users, sessions, and per-user data.

CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('admin', 'user')),
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE sessions (
    token TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expiry TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

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

DROP TABLE series_settings;
CREATE TABLE user_series_settings (
    user_id INTEGER NOT NULL,
    series_id INTEGER NOT NULL,
    sort_by TEXT NOT NULL DEFAULT 'auto',
    sort_dir TEXT NOT NULL DEFAULT 'asc',
    PRIMARY KEY (user_id, series_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
);

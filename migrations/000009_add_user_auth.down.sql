-- Create tables for users, sessions, and per-user data.
DROP TABLE IF EXISTS user_series_settings;
DROP TABLE IF EXISTS user_chapter_progress;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;


CREATE TABLE series_settings (
    series_id INTEGER PRIMARY KEY,
    sort_by TEXT NOT NULL DEFAULT 'auto',
    sort_dir TEXT NOT NULL DEFAULT 'asc',
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
);

-- A NEW migration to add a table for persistent sort settings.

CREATE TABLE series_settings (
    series_id INTEGER PRIMARY KEY,
    sort_by TEXT NOT NULL DEFAULT 'auto',
    sort_dir TEXT NOT NULL DEFAULT 'asc',
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
);

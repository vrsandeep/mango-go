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
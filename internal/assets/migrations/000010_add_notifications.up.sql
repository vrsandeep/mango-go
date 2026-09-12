PRAGMA foreign_keys = ON;

CREATE TABLE notifications (
    id INTEGER PRIMARY KEY,
    kind TEXT NOT NULL,
    series_title TEXT NOT NULL,
    chapter_title TEXT NOT NULL,
    local_folder_id INTEGER,
    local_chapter_id INTEGER,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_notifications_created_at ON notifications (created_at);

CREATE TABLE user_notification_reads (
    user_id INTEGER NOT NULL,
    notification_id INTEGER NOT NULL,
    read_at TIMESTAMP NOT NULL,
    PRIMARY KEY (user_id, notification_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (notification_id) REFERENCES notifications(id) ON DELETE CASCADE
);

ALTER TABLE subscriptions ADD COLUMN last_downloaded_at TIMESTAMP;

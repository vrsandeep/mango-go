PRAGMA foreign_keys = ON;

DROP TABLE IF EXISTS user_notification_reads;
DROP TABLE IF EXISTS notifications;
ALTER TABLE subscriptions DROP COLUMN last_downloaded_at;

-- DOWN MIGRATION: Drop all tables in reverse dependency order

-- Drop triggers first
DROP TRIGGER IF EXISTS after_insert_on_chapters;
DROP TRIGGER IF EXISTS after_delete_on_chapters;

-- Drop tables with foreign key dependencies first
DROP TABLE IF EXISTS user_chapter_progress;
DROP TABLE IF EXISTS user_series_settings;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS series_tags;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS download_queue;
DROP TABLE IF EXISTS chapters;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS series;
DROP TABLE IF EXISTS users;
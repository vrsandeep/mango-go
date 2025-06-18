DROP TRIGGER IF EXISTS after_insert_on_chapters;
DROP TRIGGER IF EXISTS after_delete_on_chapters;
DROP TRIGGER IF EXISTS after_update_on_chapters;

-- SQLite doesn't directly support DROP COLUMN in older versions.
-- The safe way is to recreate the table, but we assume
-- modern SQLite.
ALTER TABLE series DROP COLUMN total_chapters;
ALTER TABLE series DROP COLUMN read_chapters;

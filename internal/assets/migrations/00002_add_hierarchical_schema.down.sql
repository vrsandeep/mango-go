DROP TABLE IF EXISTS folder_tags;
DROP TABLE IF EXISTS folders;
-- Note: Dropping columns in SQLite is complex. This is a simplified representation.
-- In a real scenario, you'd recreate the table without these columns.
ALTER TABLE chapters DROP COLUMN folder_id;
ALTER TABLE chapters DROP COLUMN content_hash;

PRAGMA foreign_keys = ON;

-- Remove file metadata columns
DROP INDEX IF EXISTS idx_chapters_file_mtime;
ALTER TABLE chapters DROP COLUMN file_size;
ALTER TABLE chapters DROP COLUMN file_mtime;

-- Foreign key check
PRAGMA foreign_key_check;

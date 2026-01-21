PRAGMA foreign_keys = ON;

-- Add file metadata columns to chapters table for incremental scanning optimization
ALTER TABLE chapters ADD COLUMN file_mtime TIMESTAMP;
ALTER TABLE chapters ADD COLUMN file_size INTEGER;

-- Create index on file_mtime for faster queries
CREATE INDEX idx_chapters_file_mtime ON chapters(file_mtime);

-- Foreign key check
PRAGMA foreign_key_check;

-- Remove bad_files table and its indexes
DROP INDEX IF EXISTS idx_bad_files_error;
DROP INDEX IF EXISTS idx_bad_files_detected_at;
DROP INDEX IF EXISTS idx_bad_files_path;
DROP TABLE IF EXISTS bad_files;

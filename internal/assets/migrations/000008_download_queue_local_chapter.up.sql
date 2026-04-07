PRAGMA foreign_keys = ON;

ALTER TABLE download_queue ADD COLUMN local_chapter_id INTEGER;
ALTER TABLE download_queue ADD COLUMN local_folder_id INTEGER;

PRAGMA foreign_keys = ON;

ALTER TABLE download_queue DROP COLUMN local_chapter_id;
ALTER TABLE download_queue DROP COLUMN local_folder_id;

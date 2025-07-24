DROP TRIGGER IF EXISTS after_insert_on_chapters;
DROP TRIGGER IF EXISTS after_delete_on_chapters;
DROP TABLE IF EXISTS folder_tags;
DROP TABLE IF EXISTS folders;
DROP TABLE IF EXISTS user_folder_settings;

DROP TABLE IF EXISTS chapters;
CREATE TABLE chapters (
    id INTEGER PRIMARY KEY,
    series_id INTEGER NOT NULL,
    path TEXT NOT NULL UNIQUE,
    page_count INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    thumbnail TEXT,
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE
);
CREATE INDEX idx_chapters_path ON chapters (path);
-- Triggers for series chapter count
CREATE TRIGGER after_insert_on_chapters
AFTER INSERT ON chapters
BEGIN
    UPDATE series
    SET total_chapters = total_chapters + 1
    WHERE id = NEW.series_id;
END;

CREATE TRIGGER after_delete_on_chapters
AFTER DELETE ON chapters
BEGIN
    UPDATE series
    SET total_chapters = total_chapters - 1
    WHERE id = OLD.series_id;
END;

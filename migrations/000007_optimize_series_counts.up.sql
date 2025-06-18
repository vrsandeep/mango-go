-- A NEW migration to denormalize chapter counts for performance and add triggers.

-- Step 1: Add new columns to the series table
ALTER TABLE series ADD COLUMN total_chapters INTEGER NOT NULL DEFAULT 0;
ALTER TABLE series ADD COLUMN read_chapters INTEGER NOT NULL DEFAULT 0;

-- Step 2: Populate the new columns with existing data
UPDATE series
SET
    total_chapters = (SELECT COUNT(*) FROM chapters WHERE chapters.series_id = series.id),
    read_chapters = (SELECT COUNT(*) FROM chapters WHERE chapters.series_id = series.id AND chapters.read = 1);

-- Step 3: Create triggers to keep the counts updated automatically

-- Trigger for when a new chapter is inserted
CREATE TRIGGER after_insert_on_chapters
AFTER INSERT ON chapters
BEGIN
    UPDATE series
    SET total_chapters = total_chapters + 1,
        read_chapters = read_chapters + (CASE WHEN NEW.read = 1 THEN 1 ELSE 0 END)
    WHERE id = NEW.series_id;
END;

-- Trigger for when a chapter is deleted
CREATE TRIGGER after_delete_on_chapters
AFTER DELETE ON chapters
BEGIN
    UPDATE series
    SET total_chapters = total_chapters - 1,
        read_chapters = read_chapters - (CASE WHEN OLD.read = 1 THEN 1 ELSE 0 END)
    WHERE id = OLD.series_id;
END;

-- Trigger for when a chapter's read status is updated
CREATE TRIGGER after_update_on_chapters
AFTER UPDATE OF read ON chapters
WHEN OLD.read <> NEW.read
BEGIN
    UPDATE series
    SET read_chapters = read_chapters + (CASE WHEN NEW.read = 1 THEN 1 ELSE -1 END)
    WHERE id = NEW.series_id;
END;

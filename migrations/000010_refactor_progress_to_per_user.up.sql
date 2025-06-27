-- A NEW migration to finalize the shift to per-user progress by cleaning up old columns and triggers.

-- Step 1: Drop the old trigger that updated 'read_chapters' as it's no longer needed.
DROP TRIGGER IF EXISTS after_update_on_chapters;
DROP TRIGGER IF EXISTS after_insert_on_chapters;
DROP TRIGGER IF EXISTS after_delete_on_chapters;

-- Step 2: Drop the old global progress columns from the 'chapters' table.
-- In a real production environment, you would first write a script to migrate
-- this data into the new user_chapter_progress table for each user.
ALTER TABLE chapters DROP COLUMN read;
ALTER TABLE chapters DROP COLUMN progress_percent;

-- Step 3: Drop the now-redundant read_chapters column from the 'series' table.
-- total_chapters is still useful and globally correct.
ALTER TABLE series DROP COLUMN read_chapters;

-- Step 4: Recreate the insert/delete triggers to only manage 'total_chapters'.
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

-- A NEW migration to add columns for tracking reading progress to the 'chapters' table.

ALTER TABLE chapters ADD COLUMN read BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE chapters ADD COLUMN progress_percent INTEGER NOT NULL DEFAULT 0;

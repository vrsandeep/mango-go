-- A NEW migration to add TEXT columns for storing thumbnail data.

ALTER TABLE series ADD COLUMN thumbnail TEXT;
ALTER TABLE chapters ADD COLUMN thumbnail TEXT;

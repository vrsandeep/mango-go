PRAGMA foreign_keys = ON;

-- Add folder_path column to subscriptions table
ALTER TABLE subscriptions ADD COLUMN folder_path TEXT;

-- Create index for faster lookups by folder_path
CREATE INDEX idx_subscriptions_folder_path ON subscriptions (folder_path);

-- Foreign key check
PRAGMA foreign_key_check;

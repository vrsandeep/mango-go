ALTER TABLE subscriptions DROP COLUMN IF EXISTS folder_path;

DROP INDEX IF EXISTS idx_subscriptions_folder_path;

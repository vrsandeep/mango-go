DROP INDEX IF EXISTS idx_download_queue_user_id;
DROP INDEX IF EXISTS idx_subscriptions_user_id;
ALTER TABLE download_queue DROP COLUMN user_id;
ALTER TABLE subscriptions DROP COLUMN user_id;

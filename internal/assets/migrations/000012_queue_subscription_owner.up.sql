ALTER TABLE download_queue ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE subscriptions ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE SET NULL;
CREATE INDEX idx_download_queue_user_id ON download_queue(user_id);
CREATE INDEX idx_subscriptions_user_id ON subscriptions(user_id);

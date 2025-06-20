CREATE TABLE download_queue (
    id INTEGER PRIMARY KEY,
    series_title TEXT NOT NULL,
    chapter_title TEXT NOT NULL,
    chapter_identifier TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued', -- queued, in_progress, completed, failed
    progress INTEGER NOT NULL DEFAULT 0,
    message TEXT,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE subscriptions (
    id INTEGER PRIMARY KEY,
    series_title TEXT NOT NULL,
    series_identifier TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    local_series_id INTEGER, -- Nullable, links to local library if matched
    created_at TIMESTAMP NOT NULL,
    last_checked_at TIMESTAMP,
    FOREIGN KEY (local_series_id) REFERENCES series(id) ON DELETE SET NULL
);

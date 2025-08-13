-- Create bad_files table to track corrupted or invalid archive files
CREATE TABLE bad_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT UNIQUE NOT NULL,
    file_name TEXT NOT NULL,
    error TEXT NOT NULL,
    file_size INTEGER,
    detected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_checked DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_bad_files_path ON bad_files(path);
CREATE INDEX idx_bad_files_detected_at ON bad_files(detected_at);
CREATE INDEX idx_bad_files_error ON bad_files(error);

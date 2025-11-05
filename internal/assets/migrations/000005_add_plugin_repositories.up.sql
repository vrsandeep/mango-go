PRAGMA foreign_keys = ON;

-- Plugin repositories table
CREATE TABLE plugin_repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT NOT NULL UNIQUE,
    name TEXT,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Installed plugins tracking (to track which plugins came from which repository)
CREATE TABLE installed_plugins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plugin_id TEXT NOT NULL UNIQUE,
    repository_id INTEGER,
    installed_version TEXT NOT NULL,
    installed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES plugin_repositories(id) ON DELETE SET NULL
);

-- Indexes
CREATE INDEX idx_plugin_repositories_url ON plugin_repositories(url);
CREATE INDEX idx_installed_plugins_plugin_id ON installed_plugins(plugin_id);
CREATE INDEX idx_installed_plugins_repository_id ON installed_plugins(repository_id);

-- Insert default repository
INSERT INTO plugin_repositories (url, name, description)
VALUES ('https://raw.githubusercontent.com/vrsandeep/mango-go-plugins/master/repository.json', 'Mango-Go Plugins', 'Official plugin repository for mango-go');

-- Foreign key check
PRAGMA foreign_key_check;


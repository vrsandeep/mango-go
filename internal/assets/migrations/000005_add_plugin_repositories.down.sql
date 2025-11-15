PRAGMA foreign_keys = ON;

-- Drop indexes
DROP INDEX IF EXISTS idx_installed_plugins_repository_id;
DROP INDEX IF EXISTS idx_installed_plugins_plugin_id;
DROP INDEX IF EXISTS idx_plugin_repositories_url;

-- Drop tables
DROP TABLE IF EXISTS installed_plugins;
DROP TABLE IF EXISTS plugin_repositories;

-- Foreign key check
PRAGMA foreign_key_check;


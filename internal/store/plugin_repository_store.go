package store

import (
	"database/sql"
	"time"
)

// PluginRepository represents a plugin repository in the database
type PluginRepository struct {
	ID          int64
	URL         string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// InstalledPlugin represents an installed plugin tracking entry
type InstalledPlugin struct {
	ID             int64
	PluginID       string
	RepositoryID   sql.NullInt64
	InstalledVersion string
	InstalledAt    time.Time
	UpdatedAt      time.Time
}

// GetAllRepositories returns all plugin repositories
func (s *Store) GetAllRepositories() ([]*PluginRepository, error) {
	rows, err := s.db.Query(`
		SELECT id, url, name, description, created_at, updated_at
		FROM plugin_repositories
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repositories []*PluginRepository
	for rows.Next() {
		var repo PluginRepository
		err := rows.Scan(
			&repo.ID,
			&repo.URL,
			&repo.Name,
			&repo.Description,
			&repo.CreatedAt,
			&repo.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, &repo)
	}

	return repositories, rows.Err()
}

// GetRepositoryByID returns a repository by ID
func (s *Store) GetRepositoryByID(id int64) (*PluginRepository, error) {
	var repo PluginRepository
	err := s.db.QueryRow(`
		SELECT id, url, name, description, created_at, updated_at
		FROM plugin_repositories
		WHERE id = ?
	`, id).Scan(
		&repo.ID,
		&repo.URL,
		&repo.Name,
		&repo.Description,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

// GetRepositoryByURL returns a repository by URL
func (s *Store) GetRepositoryByURL(url string) (*PluginRepository, error) {
	var repo PluginRepository
	err := s.db.QueryRow(`
		SELECT id, url, name, description, created_at, updated_at
		FROM plugin_repositories
		WHERE url = ?
	`, url).Scan(
		&repo.ID,
		&repo.URL,
		&repo.Name,
		&repo.Description,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

// CreateRepository creates a new plugin repository
func (s *Store) CreateRepository(url, name, description string) (*PluginRepository, error) {
	result, err := s.db.Exec(`
		INSERT INTO plugin_repositories (url, name, description, created_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, url, name, description)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetRepositoryByID(id)
}

// UpdateRepository updates a repository
func (s *Store) UpdateRepository(id int64, name, description string) error {
	_, err := s.db.Exec(`
		UPDATE plugin_repositories
		SET name = ?, description = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, name, description, id)
	return err
}

// DeleteRepository deletes a repository
func (s *Store) DeleteRepository(id int64) error {
	_, err := s.db.Exec(`DELETE FROM plugin_repositories WHERE id = ?`, id)
	return err
}

// GetInstalledPlugin returns an installed plugin entry by plugin ID
func (s *Store) GetInstalledPlugin(pluginID string) (*InstalledPlugin, error) {
	var installed InstalledPlugin
	err := s.db.QueryRow(`
		SELECT id, plugin_id, repository_id, installed_version, installed_at, updated_at
		FROM installed_plugins
		WHERE plugin_id = ?
	`, pluginID).Scan(
		&installed.ID,
		&installed.PluginID,
		&installed.RepositoryID,
		&installed.InstalledVersion,
		&installed.InstalledAt,
		&installed.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &installed, nil
}

// CreateOrUpdateInstalledPlugin creates or updates an installed plugin entry
func (s *Store) CreateOrUpdateInstalledPlugin(pluginID string, repositoryID sql.NullInt64, version string) error {
	// Check if plugin already exists
	existing, err := s.GetInstalledPlugin(pluginID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if existing != nil {
		// Update existing entry
		_, err = s.db.Exec(`
			UPDATE installed_plugins
			SET repository_id = ?, installed_version = ?, updated_at = CURRENT_TIMESTAMP
			WHERE plugin_id = ?
		`, repositoryID, version, pluginID)
		return err
	}

	// Insert new entry
	_, err = s.db.Exec(`
		INSERT INTO installed_plugins (plugin_id, repository_id, installed_version, installed_at, updated_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, pluginID, repositoryID, version)
	return err
}

// DeleteInstalledPlugin deletes an installed plugin entry
func (s *Store) DeleteInstalledPlugin(pluginID string) error {
	_, err := s.db.Exec(`DELETE FROM installed_plugins WHERE plugin_id = ?`, pluginID)
	return err
}

// GetAllInstalledPlugins returns all installed plugin entries
func (s *Store) GetAllInstalledPlugins() ([]*InstalledPlugin, error) {
	rows, err := s.db.Query(`
		SELECT id, plugin_id, repository_id, installed_version, installed_at, updated_at
		FROM installed_plugins
		ORDER BY installed_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installed []*InstalledPlugin
	for rows.Next() {
		var inst InstalledPlugin
		err := rows.Scan(
			&inst.ID,
			&inst.PluginID,
			&inst.RepositoryID,
			&inst.InstalledVersion,
			&inst.InstalledAt,
			&inst.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		installed = append(installed, &inst)
	}

	return installed, rows.Err()
}


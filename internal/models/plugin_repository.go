package models

// RepositoryPlugin represents a plugin available in a repository
type RepositoryPlugin struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author,omitempty"`
	License     string                 `json:"license,omitempty"`
	APIVersion  string                 `json:"api_version"`
	PluginType  string                 `json:"plugin_type"`
	DownloadURL string                 `json:"download_url"`
	ManifestURL string                 `json:"manifest_url"`
	EntryPoint  string                 `json:"entry_point,omitempty"`
	Capabilities map[string]bool       `json:"capabilities,omitempty"`
	UpdatedAt   string                 `json:"updated_at,omitempty"`
}

// RepositoryManifest represents the repository.json file structure
type RepositoryManifest struct {
	Version    string            `json:"version"`
	Repository RepositoryInfo    `json:"repository"`
	Plugins    []RepositoryPlugin `json:"plugins"`
}

// RepositoryInfo contains repository metadata
type RepositoryInfo struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// PluginInstallRequest represents a request to install a plugin
type PluginInstallRequest struct {
	PluginID    string `json:"plugin_id"`
	RepositoryID int64  `json:"repository_id"`
}

// PluginUpdateInfo represents information about available plugin updates
type PluginUpdateInfo struct {
	PluginID       string `json:"plugin_id"`
	Name           string `json:"name"`
	InstalledVersion string `json:"installed_version"`
	AvailableVersion string `json:"available_version"`
	RepositoryID   int64  `json:"repository_id"`
	RepositoryName string `json:"repository_name"`
	HasUpdate      bool   `json:"has_update"`
}


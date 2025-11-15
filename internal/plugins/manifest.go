package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PluginManifest represents the plugin.json structure.
// For plugins installed from repositories, most metadata comes from repository.json.
// plugin.json only needs: id, api_version, entry_point (optional), and config.
// When installed from a repository, the manifest is enriched with repository metadata.
type PluginManifest struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name,omitempty"`        // Optional - comes from repository.json for installed plugins
	Version     string                 `json:"version,omitempty"`     // Optional - comes from repository.json for installed plugins
	Description string                 `json:"description,omitempty"` // Optional - comes from repository.json for installed plugins
	Author      string                 `json:"author,omitempty"`      // Optional - comes from repository.json for installed plugins
	License     string                 `json:"license,omitempty"`     // Optional - comes from repository.json for installed plugins
	APIVersion  string                 `json:"api_version"`
	PluginType  string                 `json:"plugin_type,omitempty"`  // Optional - defaults to "downloader"
	EntryPoint  string                 `json:"entry_point,omitempty"` // Optional - defaults to "index.js"
	Capabilities map[string]bool       `json:"capabilities,omitempty"` // Optional - comes from repository.json for installed plugins
	Config      map[string]interface{} `json:"config,omitempty"`
}

// LoadManifest loads and parses a plugin.json file.
func LoadManifest(pluginDir string) (*PluginManifest, error) {
	manifestPath := filepath.Join(pluginDir, "plugin.json")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin.json: %w", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse plugin.json: %w", err)
	}

	// Validate required fields
	if manifest.ID == "" {
		return nil, fmt.Errorf("plugin.json missing required field: id")
	}
	if manifest.APIVersion == "" {
		return nil, fmt.Errorf("plugin.json missing required field: api_version")
	}
	// Name, Version, Description, Author, License are optional - come from repository.json for installed plugins

	// Set defaults
	if manifest.PluginType == "" {
		manifest.PluginType = "downloader"
	}
	if manifest.EntryPoint == "" {
		manifest.EntryPoint = "index.js"
	}
	if manifest.Capabilities == nil {
		manifest.Capabilities = make(map[string]bool)
	}

	return &manifest, nil
}


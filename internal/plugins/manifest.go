package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PluginManifest represents the plugin.json structure.
type PluginManifest struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	License     string                 `json:"license"`
	APIVersion  string                 `json:"api_version"`
	PluginType  string                 `json:"plugin_type"`
	EntryPoint  string                 `json:"entry_point"`
	Capabilities map[string]bool       `json:"capabilities"`
	Config      map[string]interface{} `json:"config"`
	Repository  map[string]interface{} `json:"repository,omitempty"`
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
	if manifest.Name == "" {
		return nil, fmt.Errorf("plugin.json missing required field: name")
	}
	if manifest.Version == "" {
		return nil, fmt.Errorf("plugin.json missing required field: version")
	}
	if manifest.APIVersion == "" {
		return nil, fmt.Errorf("plugin.json missing required field: api_version")
	}

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


package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
)

// LoadPlugins discovers and loads all plugins from the plugins directory.
func LoadPlugins(app *core.App, pluginDir string) error {
	// Create plugins directory if it doesn't exist
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories and special directories
		if entry.Name()[0] == '.' {
			continue
		}

		pluginPath := filepath.Join(pluginDir, entry.Name())
		manifestPath := filepath.Join(pluginPath, "plugin.json")

		// Check if plugin.json exists
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			log.Printf("Skipping %s: no plugin.json found", entry.Name())
			continue
		}

		// Load plugin
		if err := LoadPlugin(app, pluginPath); err != nil {
			log.Printf("Failed to load plugin %s: %v", entry.Name(), err)
			continue
		}
	}

	return nil
}

// LoadPlugin loads a single plugin from a directory.
func LoadPlugin(app *core.App, pluginDir string) error {
	// Load manifest
	manifest, err := LoadManifest(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Only load downloader plugins for now
	if manifest.PluginType != "downloader" {
		log.Printf("Skipping plugin %s: type %s not supported yet", manifest.ID, manifest.PluginType)
		return nil
	}

	// Create runtime
	runtime, err := NewPluginRuntime(app, manifest, pluginDir)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	// Create adapter
	adapter := NewPluginProviderAdapter(runtime)

	// Register with provider registry
	providers.Register(adapter)

	log.Printf("Loaded plugin: %s (%s)", manifest.Name, manifest.ID)

	return nil
}

// UnloadPlugin unloads a plugin by ID.
func UnloadPlugin(pluginID string) error {
	// Note: goja doesn't support unloading, so we just unregister
	// In a full implementation, we'd need to track loaded plugins
	providers.Unregister(pluginID)
	return nil
}


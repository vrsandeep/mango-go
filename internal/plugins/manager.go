package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
)

const (
	// APIVersion is the current API version supported by mango-go
	APIVersion = "1.0"
)

// PluginInfo represents information about a loaded plugin
type PluginInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	License     string                 `json:"license"`
	APIVersion  string                 `json:"api_version"`
	PluginType  string                 `json:"plugin_type"`
	Capabilities map[string]bool       `json:"capabilities"`
	Path        string                 `json:"path"`
	Loaded      bool                   `json:"loaded"`
	Error       string                 `json:"error,omitempty"`
}

// PluginManager manages all loaded plugins
type PluginManager struct {
	app       *core.App
	pluginDir string
	plugins   map[string]*LoadedPlugin
	mu        sync.RWMutex
}

// LoadedPlugin represents a loaded plugin with its runtime
type LoadedPlugin struct {
	Manifest *PluginManifest
	Runtime  *PluginRuntime
	Path     string
}

var (
	globalManager PluginManagerInterface
	managerMu     sync.RWMutex
)

// NewPluginManager creates a new plugin manager
func NewPluginManager(app *core.App, pluginDir string) *PluginManager {
	return &PluginManager{
		app:       app,
		pluginDir: pluginDir,
		plugins:   make(map[string]*LoadedPlugin),
	}
}

// SetGlobalManager sets the global plugin manager instance
func SetGlobalManager(manager PluginManagerInterface) {
	managerMu.Lock()
	defer managerMu.Unlock()
	globalManager = manager
}

// GetGlobalManager returns the global plugin manager instance
func GetGlobalManager() PluginManagerInterface {
	managerMu.RLock()
	defer managerMu.RUnlock()
	return globalManager
}

// LoadPlugins discovers and loads all plugins from the plugins directory.
func (pm *PluginManager) LoadPlugins() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Create plugins directory if it doesn't exist
	if err := os.MkdirAll(pm.pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	entries, err := os.ReadDir(pm.pluginDir)
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

		pluginPath := filepath.Join(pm.pluginDir, entry.Name())
		manifestPath := filepath.Join(pluginPath, "plugin.json")

		// Check if plugin.json exists
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			log.Printf("Skipping %s: no plugin.json found", entry.Name())
			continue
		}

		// Load plugin (using internal method to avoid lock re-acquisition)
		if err := pm.loadPluginInternal(pluginPath); err != nil {
			log.Printf("Failed to load plugin %s: %v", entry.Name(), err)
			continue
		}
	}

	return nil
}

// LoadPlugin loads a single plugin from a directory.
func (pm *PluginManager) LoadPlugin(pluginDir string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.loadPluginInternal(pluginDir)
}

// loadPluginInternal loads a plugin without acquiring locks (caller must hold lock).
func (pm *PluginManager) loadPluginInternal(pluginDir string) error {
	// Load manifest
	manifest, err := LoadManifest(pluginDir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Validate API version compatibility
	if err := ValidateAPIVersion(manifest.APIVersion); err != nil {
		return fmt.Errorf("API version incompatibility: %w", err)
	}

	// Only load downloader plugins for now
	if manifest.PluginType != "downloader" {
		log.Printf("Skipping plugin %s: type %s not supported yet", manifest.ID, manifest.PluginType)
		return nil
	}

	// Check if plugin is already loaded
	if _, exists := pm.plugins[manifest.ID]; exists {
		return fmt.Errorf("plugin %s is already loaded", manifest.ID)
	}

	// Create runtime
	runtime, err := NewPluginRuntime(pm.app, manifest, pluginDir)
	if err != nil {
		return fmt.Errorf("failed to create runtime: %w", err)
	}

	// Create adapter
	adapter := NewPluginProviderAdapter(runtime)

	// Register with provider registry
	providers.Register(adapter)

	// Store loaded plugin (lock already held by caller)
	pm.plugins[manifest.ID] = &LoadedPlugin{
		Manifest: manifest,
		Runtime:  runtime,
		Path:     pluginDir,
	}

	log.Printf("Loaded plugin: %s (%s)", manifest.Name, manifest.ID)

	return nil
}

// UnloadPlugin unloads a plugin by ID.
func (pm *PluginManager) UnloadPlugin(pluginID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.plugins[pluginID]; !exists {
		return fmt.Errorf("plugin %s is not loaded", pluginID)
	}

	// Unregister from provider registry
	providers.Unregister(pluginID)

	// Remove from manager
	delete(pm.plugins, pluginID)

	log.Printf("Unloaded plugin: %s", pluginID)

	return nil
}

// ReloadPlugin reloads a plugin by ID.
func (pm *PluginManager) ReloadPlugin(pluginID string) error {
	pm.mu.Lock()
	loadedPlugin, exists := pm.plugins[pluginID]
	if !exists {
		pm.mu.Unlock()
		return fmt.Errorf("plugin %s is not loaded", pluginID)
	}
	pluginPath := loadedPlugin.Path
	pm.mu.Unlock()

	// Unload first
	if err := pm.UnloadPlugin(pluginID); err != nil {
		return err
	}

	// Load again
	return pm.LoadPlugin(pluginPath)
}

// ReloadAllPlugins reloads all loaded plugins.
func (pm *PluginManager) ReloadAllPlugins() error {
	pm.mu.RLock()
	pluginIDs := make([]string, 0, len(pm.plugins))
	for id := range pm.plugins {
		pluginIDs = append(pluginIDs, id)
	}
	pm.mu.RUnlock()

	for _, id := range pluginIDs {
		if err := pm.ReloadPlugin(id); err != nil {
			log.Printf("Failed to reload plugin %s: %v", id, err)
		}
	}

	return nil
}

// GetPluginInfo returns information about a loaded plugin.
func (pm *PluginManager) GetPluginInfo(pluginID string) (*PluginInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	loadedPlugin, exists := pm.plugins[pluginID]
	if !exists {
		return nil, false
	}

	return &PluginInfo{
		ID:          loadedPlugin.Manifest.ID,
		Name:        loadedPlugin.Manifest.Name,
		Version:     loadedPlugin.Manifest.Version,
		Description: loadedPlugin.Manifest.Description,
		Author:      loadedPlugin.Manifest.Author,
		License:     loadedPlugin.Manifest.License,
		APIVersion:  loadedPlugin.Manifest.APIVersion,
		PluginType:  loadedPlugin.Manifest.PluginType,
		Capabilities: loadedPlugin.Manifest.Capabilities,
		Path:        loadedPlugin.Path,
		Loaded:      true,
	}, true
}

// ListPlugins returns information about all loaded plugins.
func (pm *PluginManager) ListPlugins() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]PluginInfo, 0, len(pm.plugins))
	for _, loadedPlugin := range pm.plugins {
		plugins = append(plugins, PluginInfo{
			ID:          loadedPlugin.Manifest.ID,
			Name:        loadedPlugin.Manifest.Name,
			Version:     loadedPlugin.Manifest.Version,
			Description: loadedPlugin.Manifest.Description,
			Author:      loadedPlugin.Manifest.Author,
			License:     loadedPlugin.Manifest.License,
			APIVersion:  loadedPlugin.Manifest.APIVersion,
			PluginType:  loadedPlugin.Manifest.PluginType,
			Capabilities: loadedPlugin.Manifest.Capabilities,
			Path:        loadedPlugin.Path,
			Loaded:      true,
		})
	}

	return plugins
}

// ValidateAPIVersion checks if the plugin's API version is compatible
func ValidateAPIVersion(pluginAPIVersion string) error {
	// For now, we only support API version 1.0
	// In the future, we can implement semantic versioning compatibility checking
	if pluginAPIVersion != APIVersion {
		return fmt.Errorf("plugin requires API version %s, but mango-go provides %s", pluginAPIVersion, APIVersion)
	}
	return nil
}


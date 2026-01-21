package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
)

const (
	// APIVersion is the current API version supported by mango-go
	APIVersion = "1.0"
)

// PluginInfo represents information about a loaded plugin
type PluginInfo struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	Description  string          `json:"description"`
	Author       string          `json:"author"`
	License      string          `json:"license"`
	APIVersion   string          `json:"api_version"`
	PluginType   string          `json:"plugin_type"`
	Capabilities map[string]bool `json:"capabilities"`
	Path         string          `json:"path"`
	Loaded       bool            `json:"loaded"`
	Error        string          `json:"error,omitempty"`
}

// PluginManager manages all loaded plugins
type PluginManager struct {
	app               *core.App
	pluginDir         string
	plugins           map[string]*LoadedPlugin
	discoveredPlugins map[string]*DiscoveredPlugin // Plugins discovered but not loaded
	failedPlugins     map[string]string            // Map of plugin path to error message
	mu                sync.RWMutex
	unloadTimeout     time.Duration // Time after which idle plugins are unloaded
	stopChan          chan struct{} // Channel to stop the unload goroutine
	unloadStarted     bool          // Track if unload goroutine has been started
}

// DiscoveredPlugin represents a plugin that has been discovered but not loaded
type DiscoveredPlugin struct {
	Manifest *PluginManifest
	Path     string
}

// LoadedPlugin represents a loaded plugin with its runtime
type LoadedPlugin struct {
	Manifest   *PluginManifest
	Runtime    *PluginRuntime
	Path       string
	LastAccess time.Time // Last time the plugin was accessed
}

var (
	globalManager PluginManagerInterface
	managerMu     sync.RWMutex
)

// NewPluginManager creates a new plugin manager
func NewPluginManager(app *core.App, pluginDir string) *PluginManager {
	// Get unload timeout from config (default: 30 minutes)
	unloadTimeoutMinutes := app.Config().Plugins.UnloadTimeout
	if unloadTimeoutMinutes <= 0 {
		unloadTimeoutMinutes = 30 // Default to 30 minutes if not set or invalid
	}
	unloadTimeout := time.Duration(unloadTimeoutMinutes) * time.Minute

	return &PluginManager{
		app:               app,
		pluginDir:         pluginDir,
		plugins:           make(map[string]*LoadedPlugin),
		discoveredPlugins: make(map[string]*DiscoveredPlugin),
		failedPlugins:     make(map[string]string),
		unloadTimeout:     unloadTimeout,
		stopChan:          make(chan struct{}),
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

// LoadPlugins discovers all plugins from the plugins directory without loading them.
// Plugins will be loaded on-demand when first accessed.
func (pm *PluginManager) LoadPlugins() error {
	pm.mu.Lock()

	// Create plugins directory if it doesn't exist
	if err := os.MkdirAll(pm.pluginDir, 0755); err != nil {
		pm.mu.Unlock()
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	entries, err := os.ReadDir(pm.pluginDir)
	if err != nil {
		pm.mu.Unlock()
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	var discoveredPluginIDs []string
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

		// Discover plugin (load manifest only, don't load runtime)
		manifest, err := LoadManifest(pluginPath)
		if err != nil {
			log.Printf("Failed to load manifest for plugin %s: %v", entry.Name(), err)
			pm.failedPlugins[pluginPath] = err.Error()
			continue
		}

		// Validate API version compatibility
		if err := ValidateAPIVersion(manifest.APIVersion); err != nil {
			log.Printf("Skipping plugin %s: API version incompatibility: %v", entry.Name(), err)
			pm.failedPlugins[pluginPath] = err.Error()
			continue
		}

		// Only discover downloader plugins for now
		if manifest.PluginType != "downloader" {
			log.Printf("Skipping plugin %s: type %s not supported yet", entry.Name(), manifest.PluginType)
			continue
		}

		// Store discovered plugin
		pm.discoveredPlugins[manifest.ID] = &DiscoveredPlugin{
			Manifest: manifest,
			Path:     pluginPath,
		}

		// Remove from failed plugins if successfully discovered
		delete(pm.failedPlugins, pluginPath)
		discoveredPluginIDs = append(discoveredPluginIDs, manifest.ID)
	}

	// Start background goroutine to unload idle plugins (only once)
	shouldStartUnload := !pm.unloadStarted
	if shouldStartUnload {
		pm.unloadStarted = true
	}
	pm.mu.Unlock()

	// Register lazy provider wrappers (without holding lock to avoid deadlock)
	// GetInfo() on lazy adapters needs to acquire read lock
	for _, pluginID := range discoveredPluginIDs {
		lazyAdapter := NewLazyPluginProviderAdapter(pm, pluginID)
		providers.Register(lazyAdapter)
	}

	// Start unload goroutine after releasing lock
	if shouldStartUnload {
		go pm.unloadIdlePlugins()
	}

	log.Printf("Discovered %d plugin(s) (lazy loading enabled)", len(discoveredPluginIDs))
	return nil
}

// DiscoverPlugin discovers a plugin from a directory and registers it for lazy loading.
// This is used when installing new plugins or reloading plugins.
// The plugin will be loaded on-demand when first accessed.
func (pm *PluginManager) DiscoverPlugin(pluginDir string) error {
	pm.mu.Lock()

	// Load manifest to get plugin ID
	manifest, err := LoadManifest(pluginDir)
	if err != nil {
		pm.mu.Unlock()
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Validate API version compatibility
	if err := ValidateAPIVersion(manifest.APIVersion); err != nil {
		pm.mu.Unlock()
		return fmt.Errorf("API version incompatibility: %w", err)
	}

	// Only discover downloader plugins for now
	if manifest.PluginType != "downloader" {
		pm.mu.Unlock()
		return fmt.Errorf("plugin type %s not supported yet", manifest.PluginType)
	}

	// Check if plugin is already discovered or loaded
	if _, exists := pm.discoveredPlugins[manifest.ID]; exists {
		// Update path in case plugin was moved
		pm.discoveredPlugins[manifest.ID].Path = pluginDir
		pm.mu.Unlock()
		return nil // Already discovered
	}

	// Store discovered plugin
	pm.discoveredPlugins[manifest.ID] = &DiscoveredPlugin{
		Manifest: manifest,
		Path:     pluginDir,
	}

	// Remove from failed plugins if successfully discovered
	delete(pm.failedPlugins, pluginDir)
	pm.mu.Unlock()

	// Register lazy provider wrapper (without holding lock to avoid deadlock)
	lazyAdapter := NewLazyPluginProviderAdapter(pm, manifest.ID)
	providers.Register(lazyAdapter)

	log.Printf("Discovered plugin: %s (%s) - will load on first access", manifest.Name, manifest.ID)
	return nil
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

	// Note: We don't register the adapter here because the lazy adapter is already registered.
	// The lazy adapter will use this loaded plugin when needed.

	// Store loaded plugin (lock already held by caller)
	pm.plugins[manifest.ID] = &LoadedPlugin{
		Manifest:   manifest,
		Runtime:    runtime,
		Path:       pluginDir,
		LastAccess: time.Now(),
	}

	log.Printf("Loaded plugin: %s (%s)", manifest.Name, manifest.ID)

	return nil
}

// LoadPluginIfNeeded loads a plugin if it's not already loaded.
// Returns the loaded plugin adapter, or error if loading failed.
// Caller should NOT hold the lock.
func (pm *PluginManager) LoadPluginIfNeeded(pluginID string) (*PluginProviderAdapter, error) {
	pm.mu.Lock()

	// Check if already loaded
	if loadedPlugin, exists := pm.plugins[pluginID]; exists {
		// Update last access time
		loadedPlugin.LastAccess = time.Now()
		adapter := NewPluginProviderAdapter(loadedPlugin.Runtime)
		pm.mu.Unlock()
		return adapter, nil
	}

	// Check if plugin is discovered
	discovered, exists := pm.discoveredPlugins[pluginID]
	if !exists {
		pm.mu.Unlock()
		return nil, fmt.Errorf("plugin %s not found", pluginID)
	}

	pluginPath := discovered.Path
	pm.mu.Unlock()

	// Load the plugin (without holding lock to avoid deadlock)
	pm.mu.Lock()
	if err := pm.loadPluginInternal(pluginPath); err != nil {
		pm.mu.Unlock()
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	// Get the newly loaded plugin
	loadedPlugin := pm.plugins[pluginID]
	adapter := NewPluginProviderAdapter(loadedPlugin.Runtime)
	pm.mu.Unlock()

	return adapter, nil
}

// unloadIdlePlugins periodically checks for idle plugins and unloads them.
func (pm *PluginManager) unloadIdlePlugins() {
	checkInterval := pm.unloadTimeout
	// Ensure check interval is at least 1 minute to avoid issues
	if checkInterval < time.Minute {
		checkInterval = time.Minute
	}
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	log.Printf("Plugin unload checker started (checking every %v, unloading after %v of inactivity)", checkInterval, pm.unloadTimeout)

	for {
		select {
		case <-ticker.C:
			pm.mu.Lock()
			now := time.Now()
			var toUnload []string

			for pluginID, loadedPlugin := range pm.plugins {
				// Don't unload if plugin was accessed recently
				if now.Sub(loadedPlugin.LastAccess) > pm.unloadTimeout {
					toUnload = append(toUnload, pluginID)
				}
			}

			pm.mu.Unlock()

			// Unload idle plugins
			for _, pluginID := range toUnload {
				log.Printf("Unloading idle plugin: %s", pluginID)
				if err := pm.UnloadPlugin(pluginID); err != nil {
					log.Printf("Failed to unload plugin %s: %v", pluginID, err)
				}
			}

		case <-pm.stopChan:
			return
		}
	}
}

// Stop stops the plugin manager and unloads all plugins.
func (pm *PluginManager) Stop() {
	close(pm.stopChan)

	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Unload all plugins
	for pluginID := range pm.plugins {
		providers.Unregister(pluginID)
		delete(pm.plugins, pluginID)
	}

	log.Println("Plugin manager stopped")
}

// UnloadPlugin unloads a plugin by ID.
func (pm *PluginManager) UnloadPlugin(pluginID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.plugins[pluginID]; !exists {
		return fmt.Errorf("plugin %s is not loaded", pluginID)
	}

	// Don't unregister from provider registry - keep the lazy adapter registered
	// This allows the plugin to be reloaded on next access

	// Clean up runtime resources
	loadedPlugin := pm.plugins[pluginID]
	if loadedPlugin.Runtime != nil && loadedPlugin.Runtime.vm != nil {
		// VM cleanup is handled by GC, but we can explicitly clear references
		loadedPlugin.Runtime.vm = nil
	}

	// Remove from loaded plugins (but keep in discovered plugins)
	delete(pm.plugins, pluginID)

	log.Printf("Unloaded plugin: %s (will reload on next access)", pluginID)

	return nil
}

// ReloadPlugin reloads a plugin by ID.
func (pm *PluginManager) ReloadPlugin(pluginID string) error {
	pm.mu.Lock()

	// Get plugin path (from loaded or discovered)
	var pluginPath string
	if loadedPlugin, exists := pm.plugins[pluginID]; exists {
		pluginPath = loadedPlugin.Path
	} else if discovered, exists := pm.discoveredPlugins[pluginID]; exists {
		pluginPath = discovered.Path
	} else {
		pm.mu.Unlock()
		return fmt.Errorf("plugin %s not found", pluginID)
	}
	pm.mu.Unlock()

	// Unload if loaded
	if err := pm.UnloadPlugin(pluginID); err != nil {
		// Ignore error if plugin wasn't loaded
		log.Printf("Note: plugin %s was not loaded, skipping unload", pluginID)
	}

	// Re-discover (will register lazy adapter and reload on next access)
	return pm.DiscoverPlugin(pluginPath)
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

// GetPluginInfo returns information about a plugin (loaded or discovered).
func (pm *PluginManager) GetPluginInfo(pluginID string) (*PluginInfo, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Check if loaded
	if loadedPlugin, exists := pm.plugins[pluginID]; exists {
		return &PluginInfo{
			ID:           loadedPlugin.Manifest.ID,
			Name:         loadedPlugin.Manifest.Name,
			Version:      loadedPlugin.Manifest.Version,
			Description:  loadedPlugin.Manifest.Description,
			Author:       loadedPlugin.Manifest.Author,
			License:      loadedPlugin.Manifest.License,
			APIVersion:   loadedPlugin.Manifest.APIVersion,
			PluginType:   loadedPlugin.Manifest.PluginType,
			Capabilities: loadedPlugin.Manifest.Capabilities,
			Path:         loadedPlugin.Path,
			Loaded:       true,
		}, true
	}

	// Check if discovered
	if discovered, exists := pm.discoveredPlugins[pluginID]; exists {
		return &PluginInfo{
			ID:           discovered.Manifest.ID,
			Name:         discovered.Manifest.Name,
			Version:      discovered.Manifest.Version,
			Description:  discovered.Manifest.Description,
			Author:       discovered.Manifest.Author,
			License:      discovered.Manifest.License,
			APIVersion:   discovered.Manifest.APIVersion,
			PluginType:   discovered.Manifest.PluginType,
			Capabilities: discovered.Manifest.Capabilities,
			Path:         discovered.Path,
			Loaded:       false,
		}, true
	}

	return nil, false
}

// ListPlugins returns information about all discovered plugins (loaded and unloaded).
func (pm *PluginManager) ListPlugins() []PluginInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugins := make([]PluginInfo, 0, len(pm.plugins)+len(pm.discoveredPlugins)+len(pm.failedPlugins))

	// Add successfully loaded plugins
	for _, loadedPlugin := range pm.plugins {
		plugins = append(plugins, PluginInfo{
			ID:           loadedPlugin.Manifest.ID,
			Name:         loadedPlugin.Manifest.Name,
			Version:      loadedPlugin.Manifest.Version,
			Description:  loadedPlugin.Manifest.Description,
			Author:       loadedPlugin.Manifest.Author,
			License:      loadedPlugin.Manifest.License,
			APIVersion:   loadedPlugin.Manifest.APIVersion,
			PluginType:   loadedPlugin.Manifest.PluginType,
			Capabilities: loadedPlugin.Manifest.Capabilities,
			Path:         loadedPlugin.Path,
			Loaded:       true,
		})
	}

	// Add discovered but not loaded plugins
	for _, discovered := range pm.discoveredPlugins {
		// Skip if already in loaded plugins
		if _, exists := pm.plugins[discovered.Manifest.ID]; exists {
			continue
		}
		plugins = append(plugins, PluginInfo{
			ID:           discovered.Manifest.ID,
			Name:         discovered.Manifest.Name,
			Version:      discovered.Manifest.Version,
			Description:  discovered.Manifest.Description,
			Author:       discovered.Manifest.Author,
			License:      discovered.Manifest.License,
			APIVersion:   discovered.Manifest.APIVersion,
			PluginType:   discovered.Manifest.PluginType,
			Capabilities: discovered.Manifest.Capabilities,
			Path:         discovered.Path,
			Loaded:       false,
		})
	}

	// Add failed plugins (try to load manifest to get basic info)
	for pluginPath, errorMsg := range pm.failedPlugins {
		manifest, err := LoadManifest(pluginPath)
		if err != nil {
			// If we can't even load the manifest, use directory name as ID
			pluginID := filepath.Base(pluginPath)
			plugins = append(plugins, PluginInfo{
				ID:          pluginID,
				Name:        pluginID,
				Description: "Failed to load plugin",
				Path:        pluginPath,
				Loaded:      false,
				Error:       errorMsg,
			})
		} else {
			plugins = append(plugins, PluginInfo{
				ID:           manifest.ID,
				Name:         manifest.Name,
				Version:      manifest.Version,
				Description:  manifest.Description,
				Author:       manifest.Author,
				License:      manifest.License,
				APIVersion:   manifest.APIVersion,
				PluginType:   manifest.PluginType,
				Capabilities: manifest.Capabilities,
				Path:         pluginPath,
				Loaded:       false,
				Error:        errorMsg,
			})
		}
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

package plugins

import (
	"fmt"

	"github.com/vrsandeep/mango-go/internal/core"
)

// LoadPlugins discovers and loads all plugins from the plugins directory.
// Deprecated: Use PluginManager.LoadPlugins() instead. This function is kept for backwards compatibility.
func LoadPlugins(app *core.App, pluginDir string) error {
	manager := NewPluginManager(app, pluginDir)
	return manager.LoadPlugins()
}

// LoadPlugin loads a single plugin from a directory.
// Deprecated: Use PluginManager.LoadPlugin() instead. This function is kept for backwards compatibility.
func LoadPlugin(app *core.App, pluginDir string) error {
	manager := NewPluginManager(app, "")
	return manager.LoadPlugin(pluginDir)
}

// UnloadPlugin unloads a plugin by ID.
// Deprecated: Use PluginManager.UnloadPlugin() instead. This function is kept for backwards compatibility.
func UnloadPlugin(pluginID string) error {
	manager := GetGlobalManager()
	if manager == nil {
		return fmt.Errorf("plugin manager not initialized")
	}
	return manager.UnloadPlugin(pluginID)
}


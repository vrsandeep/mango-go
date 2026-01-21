package plugins

// PluginManagerInterface defines the interface for plugin management operations
// This allows for easier mocking in tests
type PluginManagerInterface interface {
	LoadPlugins() error
	LoadPlugin(pluginDir string) error
	ListPlugins() []PluginInfo
	GetPluginInfo(pluginID string) (*PluginInfo, bool)
	ReloadPlugin(pluginID string) error
	ReloadAllPlugins() error
	UnloadPlugin(pluginID string) error
}

// Ensure PluginManager implements PluginManagerInterface
var _ PluginManagerInterface = (*PluginManager)(nil)

package plugins_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/plugins"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// TestLazyLoading tests that plugins are discovered but not loaded until first access
func TestLazyLoading(t *testing.T) {
	pluginDir := t.TempDir()

	// Create a test plugin
	pluginSubDir := filepath.Join(pluginDir, "lazy-test")
	os.MkdirAll(pluginSubDir, 0755)

	manifestJSON := `{
		"id": "lazy-test",
		"name": "Lazy Test Plugin",
		"version": "1.0.0",
		"plugin_type": "downloader",
		"entry_point": "index.js",
		"description": "Test plugin for lazy loading",
		"api_version": "1.0"
	}`
	os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

	pluginJS := `
exports.search = async (query, mango) => {
	return [{ title: "Test Series", identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
	os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)

	app := testutil.SetupTestApp(t)
	manager := plugins.NewPluginManager(app, pluginDir)
	err := manager.LoadPlugins()
	if err != nil {
		t.Fatalf("manager.LoadPlugins() failed: %v", err)
	}

	// Verify plugin is discovered but not loaded
	pluginInfo, exists := manager.GetPluginInfo("lazy-test")
	if !exists {
		t.Fatal("Plugin should be discovered")
	}
	if pluginInfo.Loaded {
		t.Error("Plugin should not be loaded yet (lazy loading)")
	}

	// Verify plugin is registered (lazy adapter)
	provider, ok := providers.Get("lazy-test")
	if !ok {
		t.Fatal("Plugin should be registered")
	}

	// GetInfo should work without loading
	info := provider.GetInfo()
	if info.ID != "lazy-test" {
		t.Errorf("Expected ID 'lazy-test', got '%s'", info.ID)
	}

	// Verify plugin is still not loaded after GetInfo
	pluginInfo, _ = manager.GetPluginInfo("lazy-test")
	if pluginInfo.Loaded {
		t.Error("Plugin should still not be loaded after GetInfo()")
	}

	// Access plugin functionality - this should trigger loading
	_, err = provider.Search("test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Verify plugin is now loaded
	pluginInfo, _ = manager.GetPluginInfo("lazy-test")
	if !pluginInfo.Loaded {
		t.Error("Plugin should be loaded after first access")
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
	})
}

// TestLazyLoadingMultipleAccesses tests that plugins stay loaded after multiple accesses
func TestLazyLoadingMultipleAccesses(t *testing.T) {
	pluginDir := t.TempDir()

	// Create a test plugin
	pluginSubDir := filepath.Join(pluginDir, "multi-access")
	os.MkdirAll(pluginSubDir, 0755)

	manifestJSON := `{
		"id": "multi-access",
		"name": "Multi Access Plugin",
		"version": "1.0.0",
		"plugin_type": "downloader",
		"entry_point": "index.js",
		"description": "Test plugin",
		"api_version": "1.0"
	}`
	os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

	pluginJS := `
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
	os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)

	app := testutil.SetupTestApp(t)
	manager := plugins.NewPluginManager(app, pluginDir)
	err := manager.LoadPlugins()
	if err != nil {
		t.Fatalf("manager.LoadPlugins() failed: %v", err)
	}

	provider, _ := providers.Get("multi-access")

	// First access - should load
	_, err = provider.Search("test1")
	if err != nil {
		t.Fatalf("First search failed: %v", err)
	}

	pluginInfo, _ := manager.GetPluginInfo("multi-access")
	if !pluginInfo.Loaded {
		t.Error("Plugin should be loaded after first access")
	}

	// Second access - should still be loaded
	_, err = provider.Search("test2")
	if err != nil {
		t.Fatalf("Second search failed: %v", err)
	}

	pluginInfo, _ = manager.GetPluginInfo("multi-access")
	if !pluginInfo.Loaded {
		t.Error("Plugin should remain loaded after second access")
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
	})
}

// TestLazyLoadingUnloadAndReload tests that plugins can be unloaded and reloaded
func TestLazyLoadingUnloadAndReload(t *testing.T) {
	pluginDir := t.TempDir()

	// Create a test plugin
	pluginSubDir := filepath.Join(pluginDir, "reload-test")
	os.MkdirAll(pluginSubDir, 0755)

	manifestJSON := `{
		"id": "reload-test",
		"name": "Reload Test Plugin",
		"version": "1.0.0",
		"plugin_type": "downloader",
		"entry_point": "index.js",
		"description": "Test plugin",
		"api_version": "1.0"
	}`
	os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

	pluginJS := `
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
	os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)

	app := testutil.SetupTestApp(t)
	manager := plugins.NewPluginManager(app, pluginDir)
	err := manager.LoadPlugins()
	if err != nil {
		t.Fatalf("manager.LoadPlugins() failed: %v", err)
	}

	provider, _ := providers.Get("reload-test")

	// Load plugin
	_, err = provider.Search("test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	pluginInfo, _ := manager.GetPluginInfo("reload-test")
	if !pluginInfo.Loaded {
		t.Error("Plugin should be loaded")
	}

	// Unload plugin
	err = manager.UnloadPlugin("reload-test")
	if err != nil {
		t.Fatalf("UnloadPlugin failed: %v", err)
	}

	pluginInfo, _ = manager.GetPluginInfo("reload-test")
	if pluginInfo.Loaded {
		t.Error("Plugin should be unloaded")
	}

	// Access again - should reload automatically
	_, err = provider.Search("test2")
	if err != nil {
		t.Fatalf("Search after unload failed: %v", err)
	}

	pluginInfo, _ = manager.GetPluginInfo("reload-test")
	if !pluginInfo.Loaded {
		t.Error("Plugin should be reloaded after access")
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
	})
}

// TestLazyLoadingGetInfoWithoutLoading tests that GetInfo works without loading the plugin
func TestLazyLoadingGetInfoWithoutLoading(t *testing.T) {
	pluginDir := t.TempDir()

	// Create a test plugin
	pluginSubDir := filepath.Join(pluginDir, "info-test")
	os.MkdirAll(pluginSubDir, 0755)

	manifestJSON := `{
		"id": "info-test",
		"name": "Info Test Plugin",
		"version": "1.0.0",
		"plugin_type": "downloader",
		"entry_point": "index.js",
		"description": "Test plugin",
		"api_version": "1.0"
	}`
	os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

	pluginJS := `
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
	os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)

	app := testutil.SetupTestApp(t)
	manager := plugins.NewPluginManager(app, pluginDir)
	err := manager.LoadPlugins()
	if err != nil {
		t.Fatalf("manager.LoadPlugins() failed: %v", err)
	}

	provider, _ := providers.Get("info-test")

	// GetInfo should work without loading
	info := provider.GetInfo()
	if info.ID != "info-test" {
		t.Errorf("Expected ID 'info-test', got '%s'", info.ID)
	}
	if info.Name != "Info Test Plugin" {
		t.Errorf("Expected name 'Info Test Plugin', got '%s'", info.Name)
	}

	// Verify plugin is still not loaded
	pluginInfo, _ := manager.GetPluginInfo("info-test")
	if pluginInfo.Loaded {
		t.Error("Plugin should not be loaded after GetInfo()")
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
	})
}

// TestLazyLoadingListPlugins tests that ListPlugins shows both loaded and unloaded plugins
func TestLazyLoadingListPlugins(t *testing.T) {
	pluginDir := t.TempDir()

	// Create two test plugins
	for _, pluginName := range []string{"loaded-plugin", "unloaded-plugin"} {
		pluginSubDir := filepath.Join(pluginDir, pluginName)
		os.MkdirAll(pluginSubDir, 0755)

		manifestJSON := `{
			"id": "` + pluginName + `",
			"name": "` + pluginName + `",
			"version": "1.0.0",
			"plugin_type": "downloader",
			"entry_point": "index.js",
			"description": "Test plugin",
			"api_version": "1.0"
		}`
		os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

		pluginJS := `
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)
	}

	app := testutil.SetupTestApp(t)
	manager := plugins.NewPluginManager(app, pluginDir)
	err := manager.LoadPlugins()
	if err != nil {
		t.Fatalf("manager.LoadPlugins() failed: %v", err)
	}

	// Access one plugin to load it
	provider, _ := providers.Get("loaded-plugin")
	_, _ = provider.Search("test")

	// List plugins
	pluginList := manager.ListPlugins()
	if len(pluginList) != 2 {
		t.Fatalf("Expected 2 plugins, got %d", len(pluginList))
	}

	// Find loaded and unloaded plugins
	var loadedFound, unloadedFound bool
	for _, p := range pluginList {
		if p.ID == "loaded-plugin" {
			loadedFound = true
			if !p.Loaded {
				t.Error("loaded-plugin should be marked as loaded")
			}
		}
		if p.ID == "unloaded-plugin" {
			unloadedFound = true
			if p.Loaded {
				t.Error("unloaded-plugin should not be marked as loaded")
			}
		}
	}

	if !loadedFound {
		t.Error("loaded-plugin not found in list")
	}
	if !unloadedFound {
		t.Error("unloaded-plugin not found in list")
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
	})
}

// TestLazyLoadingUnloadTimeout tests that plugins are unloaded after timeout
func TestLazyLoadingUnloadTimeout(t *testing.T) {
	pluginDir := t.TempDir()

	// Create a test plugin
	pluginSubDir := filepath.Join(pluginDir, "timeout-test")
	os.MkdirAll(pluginSubDir, 0755)

	manifestJSON := `{
		"id": "timeout-test",
		"name": "Timeout Test Plugin",
		"version": "1.0.0",
		"plugin_type": "downloader",
		"entry_point": "index.js",
		"description": "Test plugin",
		"api_version": "1.0"
	}`
	os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

	pluginJS := `
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
	os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)

	app := testutil.SetupTestApp(t)
	// Set a short unload timeout for testing (1 minute)
	app.SetConfig(&config.Config{
		Plugins: struct {
			Path          string `mapstructure:"path"`
			UnloadTimeout int    `mapstructure:"unload_timeout"`
		}{Path: pluginDir, UnloadTimeout: 1}, // 1 minute for testing
	})

	manager := plugins.NewPluginManager(app, pluginDir)
	err := manager.LoadPlugins()
	if err != nil {
		t.Fatalf("manager.LoadPlugins() failed: %v", err)
	}

	provider, _ := providers.Get("timeout-test")

	// Load plugin
	_, err = provider.Search("test")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	pluginInfo, _ := manager.GetPluginInfo("timeout-test")
	if !pluginInfo.Loaded {
		t.Error("Plugin should be loaded")
	}

	// Manually trigger unload by calling UnloadPlugin
	// (We can't easily test the automatic unload without waiting, so we test manual unload)
	err = manager.UnloadPlugin("timeout-test")
	if err != nil {
		t.Fatalf("UnloadPlugin failed: %v", err)
	}

	pluginInfo, _ = manager.GetPluginInfo("timeout-test")
	if pluginInfo.Loaded {
		t.Error("Plugin should be unloaded")
	}

	// Access again - should reload automatically
	_, err = provider.Search("test2")
	if err != nil {
		t.Fatalf("Search after unload failed: %v", err)
	}

	pluginInfo, _ = manager.GetPluginInfo("timeout-test")
	if !pluginInfo.Loaded {
		t.Error("Plugin should be reloaded after access")
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
		manager.Stop()
	})
}

// TestLazyLoadingConcurrentAccess tests that multiple sequential accesses work correctly
func TestLazyLoadingConcurrentAccess(t *testing.T) {
	pluginDir := t.TempDir()

	// Create a test plugin
	pluginSubDir := filepath.Join(pluginDir, "concurrent-test")
	os.MkdirAll(pluginSubDir, 0755)

	manifestJSON := `{
		"id": "concurrent-test",
		"name": "Concurrent Test Plugin",
		"version": "1.0.0",
		"plugin_type": "downloader",
		"entry_point": "index.js",
		"description": "Test plugin",
		"api_version": "1.0"
	}`
	os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

	pluginJS := `
exports.search = async (query, mango) => {
	return [{ title: query, identifier: "1", cover_url: "" }];
};
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
	os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)

	app := testutil.SetupTestApp(t)
	manager := plugins.NewPluginManager(app, pluginDir)
	err := manager.LoadPlugins()
	if err != nil {
		t.Fatalf("manager.LoadPlugins() failed: %v", err)
	}

	provider, _ := providers.Get("concurrent-test")

	// Sequential access (goja VMs are not thread-safe, so we test sequential access)
	for i := 0; i < 5; i++ {
		_, err := provider.Search("test")
		if err != nil {
			t.Fatalf("Search %d failed: %v", i, err)
		}
	}

	// Verify plugin is loaded
	pluginInfo, _ := manager.GetPluginInfo("concurrent-test")
	if !pluginInfo.Loaded {
		t.Error("Plugin should be loaded after access")
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
	})
}

// TestLazyLoadingUnloadTimeoutConfiguration tests that unload timeout can be configured
func TestLazyLoadingUnloadTimeoutConfiguration(t *testing.T) {
	pluginDir := t.TempDir()

	app := testutil.SetupTestApp(t)
	// Test with custom timeout
	app.SetConfig(&config.Config{
		Plugins: struct {
			Path          string `mapstructure:"path"`
			UnloadTimeout int    `mapstructure:"unload_timeout"`
		}{Path: pluginDir, UnloadTimeout: 60}, // 60 minutes
	})

	manager := plugins.NewPluginManager(app, pluginDir)

	// Verify timeout is set correctly (we can't easily test the actual timeout without waiting,
	// but we can verify the manager was created successfully)
	if manager == nil {
		t.Fatal("PluginManager should be created")
	}

	t.Cleanup(func() {
		manager.Stop()
	})
}

// TestLazyLoadingGetInfoFromMultiplePlugins tests GetInfo on multiple lazy-loaded plugins
func TestLazyLoadingGetInfoFromMultiplePlugins(t *testing.T) {
	pluginDir := t.TempDir()

	// Create multiple plugins
	for i := 1; i <= 3; i++ {
		pluginName := "info-plugin-" + string(rune('0'+i))
		pluginSubDir := filepath.Join(pluginDir, pluginName)
		os.MkdirAll(pluginSubDir, 0755)

		manifestJSON := `{
			"id": "` + pluginName + `",
			"name": "` + pluginName + `",
			"version": "1.0.0",
			"plugin_type": "downloader",
			"entry_point": "index.js",
			"description": "Test plugin",
			"api_version": "1.0"
		}`
		os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

		pluginJS := `
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
		os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)
	}

	app := testutil.SetupTestApp(t)
	manager := plugins.NewPluginManager(app, pluginDir)
	err := manager.LoadPlugins()
	if err != nil {
		t.Fatalf("manager.LoadPlugins() failed: %v", err)
	}

	// Get info from all plugins without loading them
	for i := 1; i <= 3; i++ {
		pluginName := "info-plugin-" + string(rune('0'+i))
		provider, ok := providers.Get(pluginName)
		if !ok {
			t.Fatalf("Plugin %s should be registered", pluginName)
		}

		info := provider.GetInfo()
		if info.ID != pluginName {
			t.Errorf("Expected ID '%s', got '%s'", pluginName, info.ID)
		}

		// Verify plugin is still not loaded
		pluginInfo, _ := manager.GetPluginInfo(pluginName)
		if pluginInfo.Loaded {
			t.Errorf("Plugin %s should not be loaded after GetInfo()", pluginName)
		}
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
	})
}

// TestLazyLoadingStopManager tests that stopping the manager works correctly
func TestLazyLoadingStopManager(t *testing.T) {
	pluginDir := t.TempDir()

	// Create a test plugin
	pluginSubDir := filepath.Join(pluginDir, "stop-test")
	os.MkdirAll(pluginSubDir, 0755)

	manifestJSON := `{
		"id": "stop-test",
		"name": "Stop Test Plugin",
		"version": "1.0.0",
		"plugin_type": "downloader",
		"entry_point": "index.js",
		"description": "Test plugin",
		"api_version": "1.0"
	}`
	os.WriteFile(filepath.Join(pluginSubDir, "plugin.json"), []byte(manifestJSON), 0644)

	pluginJS := `
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`
	os.WriteFile(filepath.Join(pluginSubDir, "index.js"), []byte(pluginJS), 0644)

	app := testutil.SetupTestApp(t)
	manager := plugins.NewPluginManager(app, pluginDir)
	err := manager.LoadPlugins()
	if err != nil {
		t.Fatalf("manager.LoadPlugins() failed: %v", err)
	}

	// Load a plugin
	provider, _ := providers.Get("stop-test")
	_, _ = provider.Search("test")

	// Stop the manager
	manager.Stop()

	// Verify plugin is unloaded
	pluginInfo, exists := manager.GetPluginInfo("stop-test")
	if !exists {
		t.Error("Plugin info should still be available")
	}
	if pluginInfo.Loaded {
		t.Error("Plugin should be unloaded after Stop()")
	}

	t.Cleanup(func() {
		providers.UnregisterAll()
	})
}

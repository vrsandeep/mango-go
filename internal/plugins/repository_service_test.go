package plugins_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/plugins"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

var errTest = errors.New("test error")

// MockPluginManagerForRepo is a mock implementation of PluginManagerInterface for testing
type MockPluginManagerForRepo struct {
	mock.Mock
}

var _ plugins.PluginManagerInterface = (*MockPluginManagerForRepo)(nil)

func (m *MockPluginManagerForRepo) LoadPlugins() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPluginManagerForRepo) DiscoverPlugin(pluginDir string) error {
	args := m.Called(pluginDir)
	return args.Error(0)
}

func (m *MockPluginManagerForRepo) UnloadPlugin(pluginID string) error {
	args := m.Called(pluginID)
	return args.Error(0)
}

func (m *MockPluginManagerForRepo) ReloadPlugin(pluginID string) error {
	args := m.Called(pluginID)
	return args.Error(0)
}

func (m *MockPluginManagerForRepo) ReloadAllPlugins() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPluginManagerForRepo) GetPluginInfo(pluginID string) (*plugins.PluginInfo, bool) {
	args := m.Called(pluginID)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*plugins.PluginInfo), args.Bool(1)
}

func (m *MockPluginManagerForRepo) ListPlugins() []plugins.PluginInfo {
	args := m.Called()
	return args.Get(0).([]plugins.PluginInfo)
}

func TestRepositoryService_FetchRepository(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Setup mock server
		manifest := models.RepositoryManifest{
			Version: "1.0",
			Repository: models.RepositoryInfo{
				Name: "Test Repo",
				URL:  "https://github.com/test/repo",
			},
			Plugins: []models.RepositoryPlugin{
				{
					ID:          "test-plugin",
					Name:        "Test Plugin",
					Version:     "1.0.0",
					APIVersion:  "1.0",
					PluginType:  "downloader",
					DownloadURL: "https://raw.githubusercontent.com/test/repo/master/test-plugin/",
					ManifestURL: "https://raw.githubusercontent.com/test/repo/master/test-plugin/plugin.json",
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)
		}))
		defer server.Close()

		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())
		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		// Test fetching
		result, err := repoService.FetchRepository(server.URL)
		if err != nil {
			t.Fatalf("Failed to fetch repository: %v", err)
		}

		if result.Version != "1.0" {
			t.Errorf("Expected version '1.0', got '%s'", result.Version)
		}
		if len(result.Plugins) != 1 {
			t.Errorf("Expected 1 plugin, got %d", len(result.Plugins))
		}
		if result.Plugins[0].ID != "test-plugin" {
			t.Errorf("Expected plugin ID 'test-plugin', got '%s'", result.Plugins[0].ID)
		}
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())
		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		_, err := repoService.FetchRepository(server.URL)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
	})

	t.Run("HTTP Error", func(t *testing.T) {
		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())
		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		_, err := repoService.FetchRepository("http://invalid-url-that-does-not-exist.local")
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("HTTP 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())
		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		_, err := repoService.FetchRepository(server.URL)
		if err == nil {
			t.Error("Expected error for 404")
		}
	})
}

func TestRepositoryService_GetAvailablePlugins(t *testing.T) {
	t.Run("Success with API Version Filtering", func(t *testing.T) {
		// Create repository in database
		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())
		_, err := storeInstance.CreateRepository(
			"https://example.com/repo.json",
			"Test Repo",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		// Setup mock server
		manifest := models.RepositoryManifest{
			Version: "1.0",
			Repository: models.RepositoryInfo{
				Name: "Test Repo",
				URL:  "https://example.com/repo",
			},
			Plugins: []models.RepositoryPlugin{
				{
					ID:          "compatible-plugin",
					Name:        "Compatible Plugin",
					Version:     "1.0.0",
					APIVersion:  "1.0",
					PluginType:  "downloader",
					DownloadURL: "https://example.com/plugin/",
					ManifestURL: "https://example.com/plugin/plugin.json",
				},
				{
					ID:          "incompatible-plugin",
					Name:        "Incompatible Plugin",
					Version:     "1.0.0",
					APIVersion:  "2.0",
					PluginType:  "downloader",
					DownloadURL: "https://example.com/plugin2/",
					ManifestURL: "https://example.com/plugin2/plugin.json",
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)
		}))
		defer server.Close()

		// Create repository with server URL
		repo2, err := storeInstance.CreateRepository(
			server.URL,
			"Test Repo 2",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		plugins, err := repoService.GetAvailablePlugins(repo2.ID)
		if err != nil {
			t.Fatalf("Failed to get available plugins: %v", err)
		}

		// Should only return compatible plugins
		if len(plugins) != 1 {
			t.Errorf("Expected 1 compatible plugin, got %d", len(plugins))
		}
		if plugins[0].ID != "compatible-plugin" {
			t.Errorf("Expected compatible-plugin, got %s", plugins[0].ID)
		}
	})

	t.Run("Repository Not Found", func(t *testing.T) {
		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())
		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		_, err := repoService.GetAvailablePlugins(99999)
		if err == nil {
			t.Error("Expected error for non-existent repository")
		}
	})
}

func TestRepositoryService_InstallPlugin(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		pluginDir := t.TempDir()
		app := testutil.SetupTestApp(t)
		app.SetConfig(&config.Config{
			Plugins: struct {
				Path          string `mapstructure:"path"`
				UnloadTimeout int    `mapstructure:"unload_timeout"`
			}{Path: pluginDir, UnloadTimeout: 30},
		})

		storeInstance := store.New(app.DB())

		// Setup mock servers
		manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/repository.json" {
				manifest := models.RepositoryManifest{
					Version: "1.0",
					Repository: models.RepositoryInfo{
						Name: "Test Repo",
						URL:  "https://example.com/repo",
					},
					Plugins: []models.RepositoryPlugin{
						{
							ID:          "test-plugin",
							Name:        "Test Plugin",
							Version:     "1.0.0",
							APIVersion:  "1.0",
							PluginType:  "downloader",
							DownloadURL: "http://" + r.Host + "/plugin/",
							ManifestURL: "http://" + r.Host + "/plugin/plugin.json",
							EntryPoint:  "index.js",
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(manifest)
			} else if r.URL.Path == "/plugin/plugin.json" {
				pluginManifest := map[string]interface{}{
					"id":          "test-plugin",
					"name":        "Test Plugin",
					"version":     "1.0.0",
					"api_version": "1.0",
					"plugin_type": "downloader",
					"entry_point": "index.js",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(pluginManifest)
			} else if r.URL.Path == "/plugin/index.js" {
				w.Write([]byte(`
exports.search = async () => [];
exports.getChapters = async () => [];
exports.getPageURLs = async () => [];
`))
			}
		}))
		defer manifestServer.Close()

		// Create repository with server URL
		repo2, err := storeInstance.CreateRepository(
			manifestServer.URL+"/repository.json",
			"Test Repo",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		mockManager := new(MockPluginManagerForRepo)
		mockManager.On("GetPluginInfo", "test-plugin").Return(nil, false)
		mockManager.On("DiscoverPlugin", mock.Anything).Return(nil)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		err = repoService.InstallPlugin("test-plugin", repo2.ID)
		if err != nil {
			t.Fatalf("Failed to install plugin: %v", err)
		}

		// Verify plugin files were created
		pluginPath := filepath.Join(pluginDir, "test-plugin")
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			t.Error("Plugin directory was not created")
		}

		manifestPath := filepath.Join(pluginPath, "plugin.json")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			t.Error("plugin.json was not created")
		}

		indexPath := filepath.Join(pluginPath, "index.js")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			t.Error("index.js was not created")
		}

		mockManager.AssertExpectations(t)
	})

	t.Run("Plugin Not Found", func(t *testing.T) {
		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())
		_, err := storeInstance.CreateRepository(
			"https://example.com/repo.json",
			"Test Repo",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		manifest := models.RepositoryManifest{
			Version:    "1.0",
			Repository: models.RepositoryInfo{Name: "Test", URL: "https://example.com"},
			Plugins:    []models.RepositoryPlugin{},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)
		}))
		defer server.Close()

		repo2, err := storeInstance.CreateRepository(
			server.URL,
			"Test Repo 2",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		err = repoService.InstallPlugin("nonexistent", repo2.ID)
		if err == nil {
			t.Error("Expected error for non-existent plugin")
		}
	})

	t.Run("API Version Incompatible", func(t *testing.T) {
		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())

		manifest := models.RepositoryManifest{
			Version:    "1.0",
			Repository: models.RepositoryInfo{Name: "Test", URL: "https://example.com"},
			Plugins: []models.RepositoryPlugin{
				{
					ID:          "incompatible-plugin",
					Name:        "Incompatible",
					Version:     "1.0.0",
					APIVersion:  "2.0",
					PluginType:  "downloader",
					DownloadURL: "https://example.com/plugin/",
					ManifestURL: "https://example.com/plugin/plugin.json",
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)
		}))
		defer server.Close()

		repo, err := storeInstance.CreateRepository(
			server.URL,
			"Test Repo",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		err = repoService.InstallPlugin("incompatible-plugin", repo.ID)
		if err == nil {
			t.Error("Expected error for incompatible API version")
		}
	})

	t.Run("Load Plugin Fails", func(t *testing.T) {
		pluginDir := t.TempDir()
		app := testutil.SetupTestApp(t)
		app.SetConfig(&config.Config{
			Plugins: struct {
				Path          string `mapstructure:"path"`
				UnloadTimeout int    `mapstructure:"unload_timeout"`
			}{Path: pluginDir, UnloadTimeout: 30},
		})

		storeInstance := store.New(app.DB())

		// Setup mock servers
		manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/repository.json" {
				manifest := models.RepositoryManifest{
					Version:    "1.0",
					Repository: models.RepositoryInfo{Name: "Test", URL: "https://example.com"},
					Plugins: []models.RepositoryPlugin{
						{
							ID:          "test-plugin",
							Name:        "Test Plugin",
							Version:     "1.0.0",
							APIVersion:  "1.0",
							PluginType:  "downloader",
							DownloadURL: "http://" + r.Host + "/plugin/",
							ManifestURL: "http://" + r.Host + "/plugin/plugin.json",
							EntryPoint:  "index.js",
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(manifest)
			} else if r.URL.Path == "/plugin/plugin.json" {
				pluginManifest := map[string]interface{}{
					"id":          "test-plugin",
					"name":        "Test Plugin",
					"version":     "1.0.0",
					"api_version": "1.0",
					"plugin_type": "downloader",
					"entry_point": "index.js",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(pluginManifest)
			} else if r.URL.Path == "/plugin/index.js" {
				w.Write([]byte(""))
			}
		}))
		defer manifestServer.Close()

		repo, err := storeInstance.CreateRepository(
			manifestServer.URL+"/repository.json",
			"Test Repo",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		mockManager := new(MockPluginManagerForRepo)
		mockManager.On("GetPluginInfo", "test-plugin").Return(nil, false)
		mockManager.On("DiscoverPlugin", mock.Anything).Return(errTest)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		err = repoService.InstallPlugin("test-plugin", repo.ID)
		if err == nil {
			t.Error("Expected error when LoadPlugin fails")
		}

		// Verify plugin directory was cleaned up
		pluginPath := filepath.Join(pluginDir, "test-plugin")
		if _, err := os.Stat(pluginPath); err == nil {
			t.Error("Plugin directory should have been cleaned up on failure")
		}

		mockManager.AssertExpectations(t)
	})
}

func TestRepositoryService_CheckForUpdates(t *testing.T) {
	t.Run("Success with Updates", func(t *testing.T) {
		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())

		// Create repository
		manifest := models.RepositoryManifest{
			Version:    "1.0",
			Repository: models.RepositoryInfo{Name: "Test Repo", URL: "https://example.com"},
			Plugins: []models.RepositoryPlugin{
				{
					ID:          "plugin1",
					Name:        "Plugin 1",
					Version:     "1.1.0",
					APIVersion:  "1.0",
					PluginType:  "downloader",
					DownloadURL: "https://example.com/plugin1/",
					ManifestURL: "https://example.com/plugin1/plugin.json",
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)
		}))
		defer server.Close()

		repo, err := storeInstance.CreateRepository(
			server.URL,
			"Test Repo",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		// Create installed plugin with older version
		repoID := sql.NullInt64{Int64: repo.ID, Valid: true}
		err = storeInstance.CreateOrUpdateInstalledPlugin("plugin1", repoID, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create installed plugin: %v", err)
		}

		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		updates, err := repoService.CheckForUpdates()
		if err != nil {
			t.Fatalf("Failed to check for updates: %v", err)
		}

		if len(updates) != 1 {
			t.Errorf("Expected 1 update, got %d", len(updates))
		}

		if updates[0].PluginID != "plugin1" {
			t.Errorf("Expected plugin1, got %s", updates[0].PluginID)
		}
		if updates[0].InstalledVersion != "1.0.0" {
			t.Errorf("Expected installed version 1.0.0, got %s", updates[0].InstalledVersion)
		}
		if updates[0].AvailableVersion != "1.1.0" {
			t.Errorf("Expected available version 1.1.0, got %s", updates[0].AvailableVersion)
		}
		if !updates[0].HasUpdate {
			t.Error("Expected HasUpdate to be true")
		}
	})

	t.Run("No Updates", func(t *testing.T) {
		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())

		manifest := models.RepositoryManifest{
			Version:    "1.0",
			Repository: models.RepositoryInfo{Name: "Test Repo", URL: "https://example.com"},
			Plugins: []models.RepositoryPlugin{
				{
					ID:          "plugin1",
					Name:        "Plugin 1",
					Version:     "1.0.0",
					APIVersion:  "1.0",
					PluginType:  "downloader",
					DownloadURL: "https://example.com/plugin1/",
					ManifestURL: "https://example.com/plugin1/plugin.json",
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)
		}))
		defer server.Close()

		repo, err := storeInstance.CreateRepository(
			server.URL,
			"Test Repo",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		// Create installed plugin with same version
		repoID := sql.NullInt64{Int64: repo.ID, Valid: true}
		err = storeInstance.CreateOrUpdateInstalledPlugin("plugin1", repoID, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create installed plugin: %v", err)
		}

		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		updates, err := repoService.CheckForUpdates()
		if err != nil {
			t.Fatalf("Failed to check for updates: %v", err)
		}

		if len(updates) != 0 {
			t.Errorf("Expected 0 updates, got %d", len(updates))
		}
	})

	t.Run("Filters Incompatible Plugins", func(t *testing.T) {
		app := testutil.SetupTestApp(t)
		storeInstance := store.New(app.DB())

		manifest := models.RepositoryManifest{
			Version:    "1.0",
			Repository: models.RepositoryInfo{Name: "Test Repo", URL: "https://example.com"},
			Plugins: []models.RepositoryPlugin{
				{
					ID:          "incompatible-plugin",
					Name:        "Incompatible",
					Version:     "2.0.0",
					APIVersion:  "2.0",
					PluginType:  "downloader",
					DownloadURL: "https://example.com/plugin/",
					ManifestURL: "https://example.com/plugin/plugin.json",
				},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(manifest)
		}))
		defer server.Close()

		repo, err := storeInstance.CreateRepository(
			server.URL,
			"Test Repo",
			"",
		)
		if err != nil {
			t.Fatalf("Failed to create repository: %v", err)
		}

		repoID := sql.NullInt64{Int64: repo.ID, Valid: true}
		err = storeInstance.CreateOrUpdateInstalledPlugin("incompatible-plugin", repoID, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create installed plugin: %v", err)
		}

		mockManager := new(MockPluginManagerForRepo)
		repoService := plugins.NewRepositoryService(app, storeInstance, mockManager)

		updates, err := repoService.CheckForUpdates()
		if err != nil {
			t.Fatalf("Failed to check for updates: %v", err)
		}

		// Should not report update for incompatible plugin
		if len(updates) != 0 {
			t.Errorf("Expected 0 updates for incompatible plugin, got %d", len(updates))
		}
	})
}

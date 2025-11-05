package api_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/vrsandeep/mango-go/internal/plugins"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// MockPluginManager is a mock implementation of plugin manager methods
type MockPluginManager struct {
	mock.Mock
}

func (m *MockPluginManager) ListPlugins() []plugins.PluginInfo {
	args := m.Called()
	return args.Get(0).([]plugins.PluginInfo)
}

func (m *MockPluginManager) GetPluginInfo(pluginID string) (*plugins.PluginInfo, bool) {
	args := m.Called(pluginID)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(*plugins.PluginInfo), args.Bool(1)
}

func (m *MockPluginManager) ReloadPlugin(pluginID string) error {
	args := m.Called(pluginID)
	return args.Error(0)
}

func (m *MockPluginManager) ReloadAllPlugins() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockPluginManager) UnloadPlugin(pluginID string) error {
	args := m.Called(pluginID)
	return args.Error(0)
}

func TestPluginHandlers(t *testing.T) {
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()
	adminCookie := testutil.CookieForUser(t, server, "admin", "password", "admin")
	userCookie := testutil.CookieForUser(t, server, "user", "password", "user")

	// Create mock plugin manager
	mockManager := new(MockPluginManager)

	// Ensure mock implements the interface
	var _ plugins.PluginManagerInterface = (*MockPluginManager)(nil)

	// Set mock as global manager
	originalManager := plugins.GetGlobalManager()
	plugins.SetGlobalManager(mockManager)

	t.Cleanup(func() {
		if originalManager != nil {
			plugins.SetGlobalManager(originalManager)
		} else {
			plugins.SetGlobalManager(nil)
		}
		mockManager.AssertExpectations(t)
	})

	testPlugin := plugins.PluginInfo{
		ID:          "test-plugin",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		Description: "Test plugin",
		Author:      "Test Author",
		License:     "MIT",
		APIVersion:  "1.0",
		PluginType:  "downloader",
		Capabilities: map[string]bool{
			"search":    true,
			"chapters":  true,
			"download":  true,
		},
		Path:   "/test/path",
		Loaded: true,
	}

	t.Run("List Plugins - Authenticated", func(t *testing.T) {
		mockManager.ExpectedCalls = nil // Clear previous expectations
		mockManager.On("ListPlugins").Return([]plugins.PluginInfo{testPlugin})

		req, _ := http.NewRequest("GET", "/api/plugins", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", status, rr.Body.String())
		}

		var pluginList []plugins.PluginInfo
		if err := json.Unmarshal(rr.Body.Bytes(), &pluginList); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(pluginList) != 1 {
			t.Errorf("Expected 1 plugin, got %d", len(pluginList))
		}

		if pluginList[0].ID != "test-plugin" {
			t.Errorf("Expected ID 'test-plugin', got '%s'", pluginList[0].ID)
		}
		if pluginList[0].Name != "Test Plugin" {
			t.Errorf("Expected name 'Test Plugin', got '%s'", pluginList[0].Name)
		}
		if !pluginList[0].Loaded {
			t.Error("Expected plugin to be loaded")
		}
	})

	t.Run("List Plugins - Empty List", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		mockManager.On("ListPlugins").Return([]plugins.PluginInfo{})

		req, _ := http.NewRequest("GET", "/api/plugins", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", status, rr.Body.String())
		}

		var pluginList []plugins.PluginInfo
		if err := json.Unmarshal(rr.Body.Bytes(), &pluginList); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(pluginList) != 0 {
			t.Errorf("Expected 0 plugins, got %d", len(pluginList))
		}
	})

	t.Run("List Plugins - Unauthenticated", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/plugins", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", status)
		}
	})

	t.Run("Get Plugin Info - Success", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		mockManager.On("GetPluginInfo", "test-plugin").Return(&testPlugin, true)

		req, _ := http.NewRequest("GET", "/api/plugins/test-plugin", nil)
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", status, rr.Body.String())
		}

		var pluginInfo plugins.PluginInfo
		if err := json.Unmarshal(rr.Body.Bytes(), &pluginInfo); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if pluginInfo.ID != "test-plugin" {
			t.Errorf("Expected ID 'test-plugin', got '%s'", pluginInfo.ID)
		}
		if pluginInfo.Name != "Test Plugin" {
			t.Errorf("Expected name 'Test Plugin', got '%s'", pluginInfo.Name)
		}
	})

	t.Run("Get Plugin Info - Not Found", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		mockManager.On("GetPluginInfo", "nonexistent").Return(nil, false)

		req, _ := http.NewRequest("GET", "/api/plugins/nonexistent", nil)
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", status)
		}
	})

	t.Run("Reload Plugin - Admin Only - Success", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		mockManager.On("ReloadPlugin", "test-plugin").Return(nil)

		req, _ := http.NewRequest("POST", "/api/admin/plugins/test-plugin/reload", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", status, rr.Body.String())
		}

		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		expectedMessage := "Plugin test-plugin reloaded successfully"
		if response["message"] != expectedMessage {
			t.Errorf("Expected message '%s', got '%s'", expectedMessage, response["message"])
		}
	})

	t.Run("Reload Plugin - Admin Only - Error", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		mockManager.On("ReloadPlugin", "test-plugin").Return(errors.New("reload failed"))

		req, _ := http.NewRequest("POST", "/api/admin/plugins/test-plugin/reload", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d: %s", status, rr.Body.String())
		}
	})

	t.Run("Reload Plugin - Non-Admin", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/admin/plugins/test-plugin/reload", nil)
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", status)
		}
	})

	t.Run("Reload All Plugins - Admin Only - Success", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		mockManager.On("ReloadAllPlugins").Return(nil)

		req, _ := http.NewRequest("POST", "/api/admin/plugins/reload", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", status, rr.Body.String())
		}

		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["message"] != "All plugins reloaded successfully" {
			t.Errorf("Expected message 'All plugins reloaded successfully', got '%s'", response["message"])
		}
	})

	t.Run("Reload All Plugins - Admin Only - Error", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		mockManager.On("ReloadAllPlugins").Return(errors.New("reload all failed"))

		req, _ := http.NewRequest("POST", "/api/admin/plugins/reload", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d: %s", status, rr.Body.String())
		}
	})

	t.Run("Unload Plugin - Admin Only - Success", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		mockManager.On("UnloadPlugin", "test-plugin").Return(nil)
		// After unload, GetPluginInfo should return not found
		mockManager.On("GetPluginInfo", "test-plugin").Return(nil, false)

		req, _ := http.NewRequest("DELETE", "/api/admin/plugins/test-plugin", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", status, rr.Body.String())
		}

		var response map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		expectedMessage := "Plugin test-plugin unloaded successfully"
		if response["message"] != expectedMessage {
			t.Errorf("Expected message '%s', got '%s'", expectedMessage, response["message"])
		}

		// Verify plugin is unloaded
		req2, _ := http.NewRequest("GET", "/api/plugins/test-plugin", nil)
		req2.AddCookie(userCookie)
		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, req2)

		if status := rr2.Code; status != http.StatusNotFound {
			t.Errorf("Expected status 404 after unload, got %d", status)
		}
	})

	t.Run("Unload Plugin - Admin Only - Error", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		mockManager.On("UnloadPlugin", "test-plugin").Return(errors.New("unload failed"))

		req, _ := http.NewRequest("DELETE", "/api/admin/plugins/test-plugin", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d: %s", status, rr.Body.String())
		}
	})

	t.Run("List Plugins - Manager Not Initialized", func(t *testing.T) {
		// Temporarily set manager to nil
		plugins.SetGlobalManager(nil)
		defer func() {
			plugins.SetGlobalManager(mockManager)
		}()

		req, _ := http.NewRequest("GET", "/api/plugins", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d: %s", status, rr.Body.String())
		}
	})

	t.Run("Get Plugin Info - Manager Not Initialized", func(t *testing.T) {
		// Temporarily set manager to nil
		plugins.SetGlobalManager(nil)
		defer func() {
			plugins.SetGlobalManager(mockManager)
		}()

		req, _ := http.NewRequest("GET", "/api/plugins/test-plugin", nil)
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d: %s", status, rr.Body.String())
		}
	})
}

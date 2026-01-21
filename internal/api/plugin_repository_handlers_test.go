package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/plugins"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestPluginRepositoryHandlers(t *testing.T) {
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()
	adminCookie := testutil.CookieForUser(t, server, "admin", "password", "admin")
	userCookie := testutil.CookieForUser(t, server, "user", "password", "user")

	// Create mock plugin manager
	mockManager := new(MockPluginManager)

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

	t.Run("List Repositories - Authenticated", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		req, _ := http.NewRequest("GET", "/api/plugin-repositories", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var repos []*models.PluginRepository
		if err := json.NewDecoder(rr.Body).Decode(&repos); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Should have at least the default repository
		if len(repos) == 0 {
			t.Error("Expected at least one repository (default)")
		}
	})

	t.Run("List Repositories - Unauthenticated", func(t *testing.T) {
		mockManager.ExpectedCalls = nil
		req, _ := http.NewRequest("GET", "/api/plugin-repositories", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("Create Repository - Admin", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		// Setup mock server for repository validation
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			manifest := models.RepositoryManifest{
				Version:    "1.0",
				Repository: models.RepositoryInfo{Name: "Test Repo", URL: "https://example.com"},
				Plugins:    []models.RepositoryPlugin{},
			}
			json.NewEncoder(w).Encode(manifest)
		}))
		defer mockServer.Close()

		reqBody := map[string]string{
			"url":         mockServer.URL,
			"name":        "Test Repository",
			"description": "Test description",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}

		var repo *models.PluginRepository
		if err := json.NewDecoder(rr.Body).Decode(&repo); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if repo.URL != mockServer.URL {
			t.Errorf("Expected URL %s, got %s", mockServer.URL, repo.URL)
		}
		if repo.Name != "Test Repository" {
			t.Errorf("Expected name 'Test Repository', got '%s'", repo.Name)
		}
	})

	t.Run("Create Repository - Invalid URL", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		reqBody := map[string]string{
			"url":  "http://invalid-url-that-does-not-exist.local",
			"name": "Invalid Repo",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Create Repository - Non-Admin", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		reqBody := map[string]string{
			"url":  "https://example.com/repo.json",
			"name": "Test Repo",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusForbidden {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
		}
	})

	t.Run("Get Repository Plugins", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		// Setup mock server
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
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
						DownloadURL: "https://example.com/plugin/",
						ManifestURL: "https://example.com/plugin/plugin.json",
					},
				},
			}
			json.NewEncoder(w).Encode(manifest)
		}))
		defer mockServer.Close()

		// Create repository first
		createReqBody := map[string]string{
			"url":  mockServer.URL,
			"name": "Test Repo",
		}
		createBody, _ := json.Marshal(createReqBody)
		createReq, _ := http.NewRequest("POST", "/api/admin/plugin-repositories", bytes.NewBuffer(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.AddCookie(adminCookie)
		createRR := httptest.NewRecorder()
		router.ServeHTTP(createRR, createReq)

		var createdRepo *models.PluginRepository
		if err := json.NewDecoder(createRR.Body).Decode(&createdRepo); err != nil {
			t.Fatalf("Failed to decode created repo: %v", err)
		}

		// Now get plugins using the created repo ID
		reqURL := "/api/plugin-repositories/" + fmt.Sprintf("%d", createdRepo.ID) + "/plugins"
		req, _ := http.NewRequest("GET", reqURL, nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			body := rr.Body.String()
			t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusOK, body)
		}

		if rr.Code == http.StatusOK {
			var availablePlugins []models.RepositoryPlugin
			if err := json.NewDecoder(rr.Body).Decode(&availablePlugins); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if len(availablePlugins) == 0 {
				t.Error("Expected at least one plugin")
			}
		}
	})

	t.Run("Delete Repository - Admin", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		// Create repository first
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			manifest := models.RepositoryManifest{
				Version:    "1.0",
				Repository: models.RepositoryInfo{Name: "Test", URL: "https://example.com"},
				Plugins:    []models.RepositoryPlugin{},
			}
			json.NewEncoder(w).Encode(manifest)
		}))
		defer mockServer.Close()

		createReqBody := map[string]string{
			"url":  mockServer.URL,
			"name": "To Delete",
		}
		createBody, _ := json.Marshal(createReqBody)
		createReq, _ := http.NewRequest("POST", "/api/admin/plugin-repositories", bytes.NewBuffer(createBody))
		createReq.Header.Set("Content-Type", "application/json")
		createReq.AddCookie(adminCookie)
		createRR := httptest.NewRecorder()
		router.ServeHTTP(createRR, createReq)

		var createdRepo *models.PluginRepository
		json.NewDecoder(createRR.Body).Decode(&createdRepo)

		// Delete repository (use the created repo's ID)
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/admin/plugin-repositories/%d", createdRepo.ID), nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("Delete Repository - Default Repository Protected", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		// Get the default repository (ID 1 from migration)
		req, _ := http.NewRequest("DELETE", "/api/admin/plugin-repositories/1", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusForbidden {
			body := rr.Body.String()
			t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, http.StatusForbidden, body)
		}

		var errorResp map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&errorResp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if errorResp["error"] != "Cannot delete the default repository" {
			t.Errorf("Expected error message about default repository, got: %s", errorResp["error"])
		}
	})

	t.Run("Check Updates - Admin", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories/check-updates", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var updates []models.PluginUpdateInfo
		if err := json.NewDecoder(rr.Body).Decode(&updates); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Updates can be empty if no plugins are installed
		// Just verify it's a valid array
		if updates == nil {
			t.Error("Expected updates to be an array, got nil")
		}
	})

	t.Run("Create Repository - Duplicate URL", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		// Setup mock server for repository validation
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			manifest := models.RepositoryManifest{
				Version:    "1.0",
				Repository: models.RepositoryInfo{Name: "Test Repo", URL: "https://example.com"},
				Plugins:    []models.RepositoryPlugin{},
			}
			json.NewEncoder(w).Encode(manifest)
		}))
		defer mockServer.Close()

		// Create first repository
		reqBody1 := map[string]string{
			"url":  mockServer.URL,
			"name": "First Repository",
		}
		body1, _ := json.Marshal(reqBody1)
		req1, _ := http.NewRequest("POST", "/api/admin/plugin-repositories", bytes.NewBuffer(body1))
		req1.Header.Set("Content-Type", "application/json")
		req1.AddCookie(adminCookie)
		rr1 := httptest.NewRecorder()
		router.ServeHTTP(rr1, req1)

		if rr1.Code != http.StatusCreated {
			t.Fatalf("Failed to create first repository: got status %d", rr1.Code)
		}

		// Try to create duplicate
		reqBody2 := map[string]string{
			"url":  mockServer.URL,
			"name": "Duplicate Repository",
		}
		body2, _ := json.Marshal(reqBody2)
		req2, _ := http.NewRequest("POST", "/api/admin/plugin-repositories", bytes.NewBuffer(body2))
		req2.Header.Set("Content-Type", "application/json")
		req2.AddCookie(adminCookie)
		rr2 := httptest.NewRecorder()
		router.ServeHTTP(rr2, req2)

		if status := rr2.Code; status != http.StatusConflict {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}

		var errorResp map[string]string
		if err := json.NewDecoder(rr2.Body).Decode(&errorResp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if errorResp["error"] != "A repository with this URL already exists" {
			t.Errorf("Expected humanized error message, got: %s", errorResp["error"])
		}
	})

	t.Run("Create Repository - Missing URL", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		reqBody := map[string]string{
			"name": "Repository Without URL",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Get Repository Plugins - Invalid Repository ID", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		req, _ := http.NewRequest("GET", "/api/plugin-repositories/invalid-id/plugins", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Get Repository Plugins - Non-existent Repository", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		req, _ := http.NewRequest("GET", "/api/plugin-repositories/99999/plugins", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})

	t.Run("Delete Repository - Invalid Repository ID", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		req, _ := http.NewRequest("DELETE", "/api/admin/plugin-repositories/invalid-id", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Install Plugin - Missing Plugin ID", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		reqBody := map[string]interface{}{
			"repository_id": 1,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories/install", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Update Plugin - Missing Plugin ID", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		reqBody := map[string]interface{}{
			"repository_id": 1,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories/update", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Update Plugin - Plugin Not Installed", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		reqBody := models.PluginInstallRequest{
			PluginID:     "non-existent-plugin",
			RepositoryID: 1,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories/update", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("Check Updates - Manager Not Initialized", func(t *testing.T) {
		// Temporarily set manager to nil
		originalManager := plugins.GetGlobalManager()
		plugins.SetGlobalManager(nil)

		t.Cleanup(func() {
			if originalManager != nil {
				plugins.SetGlobalManager(originalManager)
			} else {
				plugins.SetGlobalManager(mockManager)
			}
		})

		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories/check-updates", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})

	t.Run("Update Plugin - Manager Not Initialized", func(t *testing.T) {
		// Temporarily set manager to nil
		originalManager := plugins.GetGlobalManager()
		plugins.SetGlobalManager(nil)

		t.Cleanup(func() {
			if originalManager != nil {
				plugins.SetGlobalManager(originalManager)
			} else {
				plugins.SetGlobalManager(mockManager)
			}
		})

		reqBody := models.PluginInstallRequest{
			PluginID:     "test-plugin",
			RepositoryID: 1,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories/update", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})

	t.Run("Install Plugin - Manager Not Initialized", func(t *testing.T) {
		// Temporarily set manager to nil
		originalManager := plugins.GetGlobalManager()
		plugins.SetGlobalManager(nil)

		t.Cleanup(func() {
			if originalManager != nil {
				plugins.SetGlobalManager(originalManager)
			} else {
				plugins.SetGlobalManager(mockManager)
			}
		})

		reqBody := models.PluginInstallRequest{
			PluginID:     "test-plugin",
			RepositoryID: 1,
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories/install", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})

	t.Run("Get Repository Plugins - Manager Not Initialized", func(t *testing.T) {
		// Temporarily set manager to nil
		originalManager := plugins.GetGlobalManager()
		plugins.SetGlobalManager(nil)

		t.Cleanup(func() {
			if originalManager != nil {
				plugins.SetGlobalManager(originalManager)
			} else {
				plugins.SetGlobalManager(mockManager)
			}
		})

		req, _ := http.NewRequest("GET", "/api/plugin-repositories/1/plugins", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})

	t.Run("Create Repository - Manager Not Initialized", func(t *testing.T) {
		// Temporarily set manager to nil
		originalManager := plugins.GetGlobalManager()
		plugins.SetGlobalManager(nil)

		t.Cleanup(func() {
			if originalManager != nil {
				plugins.SetGlobalManager(originalManager)
			} else {
				plugins.SetGlobalManager(mockManager)
			}
		})

		reqBody := map[string]string{
			"url":  "https://example.com/repo.json",
			"name": "Test Repo",
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusInternalServerError {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusInternalServerError)
		}
	})

	t.Run("Create Repository - Invalid JSON", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Install Plugin - Invalid JSON", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories/install", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Update Plugin - Invalid JSON", func(t *testing.T) {
		mockManager.ExpectedCalls = nil

		req, _ := http.NewRequest("POST", "/api/admin/plugin-repositories/update", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

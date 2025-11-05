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
	"github.com/vrsandeep/mango-go/internal/store"
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

		var repos []store.PluginRepository
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

		var repo store.PluginRepository
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

		var createdRepo store.PluginRepository
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

		var createdRepo store.PluginRepository
		json.NewDecoder(createRR.Body).Decode(&createdRepo)

		// Delete repository (use the created repo's ID)
		req, _ := http.NewRequest("DELETE", "/api/admin/plugin-repositories/1", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
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
}


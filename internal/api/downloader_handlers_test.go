package api_test

// A test file for the downloader API endpoints.
import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/downloader/providers/mockadex"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

// setupTestServerWithProviders initializes a full core.App and api.Server for integration testing.
func SetupTestServerWithProviders(t *testing.T) (*api.Server, *sql.DB) {
	t.Helper()
	hub := websocket.NewHub()
	go hub.Run()

	db := testutil.SetupTestDB(t)

	app := &core.App{Version: "test"}
	app.SetConfig(&config.Config{
		Library: struct {
			Path string `mapstructure:"path"`
		}{Path: t.TempDir()},
	})
	app.SetDB(db)
	app.SetWsHub(hub)

	// Register providers for the test environment
	providers.Register(mockadex.New())

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Failed to close database: %v", err)
		}
		// Close the WebSocket hub
		// hub.Close()

		// Unregister all providers
		providers.UnregisterAll()

	})

	return api.NewServer(app), db
}

func TestDownloaderHandlers(t *testing.T) {
	server, db := SetupTestServerWithProviders(t)
	router := server.Router()

	t.Run("List Providers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/providers", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var providerList []models.ProviderInfo
		if err := json.Unmarshal(rr.Body.Bytes(), &providerList); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if len(providerList) < 1 || providerList[0].ID != "mockadex" {
			t.Errorf("handler returned incorrect provider list: got %+v", providerList)
		}
	})

	t.Run("Provider Search", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/providers/mockadex/search?q=manga", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		var results []models.SearchResult
		json.Unmarshal(rr.Body.Bytes(), &results)
		if len(results) != 10 {
			t.Errorf("Expected 10 search results, got %d", len(results))
		}
	})

	t.Run("Provider Get Chapters", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/providers/mockadex/series/mock-series-1", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		var results []models.ChapterResult
		json.Unmarshal(rr.Body.Bytes(), &results)
		if len(results) != 25 {
			t.Errorf("Expected 25 chapter results, got %d", len(results))
		}
	})

	t.Run("Add Chapters to Queue", func(t *testing.T) {
		payload := api.ChapterQueuePayload{
			SeriesTitle: "Test Manga",
			ProviderID:  "mockadex",
			Chapters: []models.ChapterResult{
				{Identifier: "ch1", Title: "Chapter 1"},
				{Identifier: "ch2", Title: "Chapter 2"},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/downloads/queue", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusAccepted {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
		}

		var count int
		db.QueryRow("SELECT COUNT(*) FROM download_queue").Scan(&count)
		if count != 2 {
			t.Errorf("Expected 2 items in queue, but found %d", count)
		}
	})
}

func TestHandleGetDownloadQueue(t *testing.T) {
	server, db := SetupTestServerWithProviders(t)
	router := server.Router()

	// Add a dummy item to the queue
	db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at) VALUES ('Test', 'Ch. 1', 'id1', 'mockadex', ?)", time.Now())

	req, _ := http.NewRequest("GET", "/api/downloads/queue", nil)
	req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var items []*models.DownloadQueueItem
	json.Unmarshal(rr.Body.Bytes(), &items)
	if len(items) != 1 {
		t.Errorf("Expected 1 item in queue, got %d", len(items))
	}
}

func TestHandleQueueAction(t *testing.T) {
	server, db := SetupTestServerWithProviders(t)
	router := server.Router()

	t.Run("Test pause all action", func(t *testing.T) {
		// Add a dummy item to the queue
		db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at) VALUES ('Test', 'Ch. 1', 'id1', 'mockadex', ?)", time.Now())
		payload := struct {
			Action string `json:"action"`
		}{Action: "pause_all"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/downloads/action", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["status"] != "success" {
			t.Errorf("Expected success status, got %s", response["status"])
		}
		// Check db
		var count int
		db.QueryRow("SELECT COUNT(*) FROM download_queue WHERE status = 'paused'").Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 paused item, got %d", count)
		}
		db.Exec("DELETE FROM download_queue") // Clean up after test
	})

	t.Run("Test resume all action", func(t *testing.T) {
		// Add a dummy item to the queue
		db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Test', 'Ch. 1', 'id1', 'mockadex', ?, ?)", time.Now(), "paused")
		payload := struct {
			Action string `json:"action"`
		}{Action: "resume_all"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/downloads/action", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["status"] != "success" {
			t.Errorf("Expected success status, got %s", response["status"])
		}
		// Check db
		var count int
		db.QueryRow("SELECT COUNT(*) FROM download_queue WHERE status = 'queued'").Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 queued item, got %d", count)
		}
		db.Exec("DELETE FROM download_queue") // Clean up after test
	})

	t.Run("Test retry failed items", func(t *testing.T) {
		// Add a failed item to the queue
		db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Test', 'Ch. 2', 'id2', 'mockadex', ?, 'failed')", time.Now())

		payload := struct {
			Action string `json:"action"`
		}{Action: "retry_failed"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/downloads/action", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["status"] != "success" {
			t.Errorf("Expected success status, got %s", response["status"])
		}
		// Check db
		var count int
		db.QueryRow("SELECT COUNT(*) FROM download_queue WHERE status = 'queued'").Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 queued item after reset, got %d", count)
		}
		db.Exec("DELETE FROM download_queue") // Clean up after test
	})

	t.Run("Test delete completed items", func(t *testing.T) {
		// Add a completed item to the queue
		db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Test', 'Ch. 3', 'id3', 'mockadex', ?, 'completed')", time.Now())

		payload := struct {
			Action string `json:"action"`
		}{Action: "delete_completed"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/downloads/action", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["status"] != "success" {
			t.Errorf("Expected success status, got %s", response["status"])
		}
		// Check db
		var count int
		db.QueryRow("SELECT COUNT(*) FROM download_queue WHERE status = 'completed'").Scan(&count)
		if count != 0 {
			t.Errorf("Expected 0 completed items after deletion, got %d", count)
		}
		db.Exec("DELETE FROM download_queue") // Clean up after test
	})

	t.Run("Empty Queue", func(t *testing.T) {
		// Add items with various statuses
		db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 1', 'id1', 'p1', ?, 'queued')", time.Now())
		db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 2', 'id2', 'p1', ?, 'failed')", time.Now())
		db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Manga', 'Ch 3', 'id3', 'p1', ?, 'completed')", time.Now())

		payload := `{"action": "empty_queue"}`
		req, _ := http.NewRequest("POST", "/api/downloads/action", bytes.NewBufferString(payload))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify only the 'completed' item remains
		var count int
		db.QueryRow("SELECT COUNT(*) FROM download_queue").Scan(&count)
		if count != 1 {
			t.Errorf("Expected 1 item to remain in queue, but found %d", count)
		}

		var status string
		db.QueryRow("SELECT status FROM download_queue").Scan(&status)
		if status != "completed" {
			t.Errorf("The remaining item should have status 'completed', but has '%s'", status)
		}
	})

	t.Run("Test invalid action", func(t *testing.T) {
		payload := struct {
			Action string `json:"action"`
		}{Action: "invalid_action"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/downloads/action", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["error"] != "Invalid action" {
			t.Errorf("Expected error message 'Invalid action', got %s", response["error"])
		}
	})
}

func TestHandleQueueItemAction(t *testing.T) {
	server, db := SetupTestServerWithProviders(t)
	router := server.Router()

	t.Run("Test delete item action", func(t *testing.T) {
		// Add a dummy item to the queue
		db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at) VALUES ('Test', 'Ch. 1', 'id1', 'mockadex', ?)", time.Now())

		// Get the inserted item ID
		var itemID int64
		db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test'").Scan(&itemID)

		payload := struct {
			Action string `json:"action"`
		}{Action: "delete"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/downloads/queue/%d/action", itemID), bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["status"] != "success" {
			t.Errorf("Expected success status, got %s", response["status"])
		}

		// Verify item was deleted
		var count int
		db.QueryRow("SELECT COUNT(*) FROM download_queue WHERE id = ?", itemID).Scan(&count)
		if count != 0 {
			t.Errorf("Expected 0 items after deletion, got %d", count)
		}
	})

	t.Run("Test pause item action", func(t *testing.T) {
		// Add a dummy item to the queue
		db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Test', 'Ch. 2', 'id2', 'mockadex', ?, 'queued')", time.Now())

		// Get the inserted item ID
		var itemID int64
		db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test' AND chapter_title = 'Ch. 2'").Scan(&itemID)

		payload := struct {
			Action string `json:"action"`
		}{Action: "pause"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/downloads/queue/%d/action", itemID), bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["status"] != "success" {
			t.Errorf("Expected success status, got %s", response["status"])
		}

		// Verify item was paused
		var status string
		db.QueryRow("SELECT status FROM download_queue WHERE id = ?", itemID).Scan(&status)
		if status != "paused" {
			t.Errorf("Expected status 'paused', got %s", status)
		}

		// Clean up
		db.Exec("DELETE FROM download_queue WHERE id = ?", itemID)
	})

	t.Run("Test resume item action", func(t *testing.T) {
		// Add a paused item to the queue
		db.Exec("INSERT INTO download_queue (series_title, chapter_title, chapter_identifier, provider_id, created_at, status) VALUES ('Test', 'Ch. 3', 'id3', 'mockadex', ?, 'paused')", time.Now())

		// Get the inserted item ID
		var itemID int64
		db.QueryRow("SELECT id FROM download_queue WHERE series_title = 'Test' AND chapter_title = 'Ch. 3'").Scan(&itemID)

		payload := struct {
			Action string `json:"action"`
		}{Action: "resume"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", fmt.Sprintf("/api/downloads/queue/%d/action", itemID), bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["status"] != "success" {
			t.Errorf("Expected success status, got %s", response["status"])
		}

		// Verify item was resumed
		var status string
		db.QueryRow("SELECT status FROM download_queue WHERE id = ?", itemID).Scan(&status)
		if status != "queued" {
			t.Errorf("Expected status 'queued', got %s", status)
		}

		// Clean up
		db.Exec("DELETE FROM download_queue WHERE id = ?", itemID)
	})

	t.Run("Test invalid item action", func(t *testing.T) {
		payload := struct {
			Action string `json:"action"`
		}{Action: "invalid_action"}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/downloads/queue/1/action", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
		}

		var response map[string]string
		json.Unmarshal(rr.Body.Bytes(), &response)
		if response["error"] != "Invalid action" {
			t.Errorf("Expected error message 'Invalid action', got %s", response["error"])
		}
	})
}

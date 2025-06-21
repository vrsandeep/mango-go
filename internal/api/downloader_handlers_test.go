package api

// A test file for the downloader API endpoints.
import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/downloader/providers/mockadex"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

// setupTestServerWithProviders initializes a full core.App and api.Server for integration testing.
func setupTestServerWithProviders(t *testing.T) *Server {
	t.Helper()
	hub := websocket.NewHub()
	go hub.Run()

	db := testutil.SetupTestDB(t)

	app := &core.App{
		Config: &config.Config{
			Library: struct {
				Path string `mapstructure:"path"`
			}{Path: t.TempDir()},
		},
		DB:    db,
		WsHub: hub,
	}

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

	return NewServer(app)
}

func TestDownloaderHandlers(t *testing.T) {
	server := setupTestServerWithProviders(t)
	router := server.Router()

	t.Run("List Providers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/providers", nil)
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
		payload := ChapterQueuePayload{
			SeriesTitle: "Test Manga",
			ProviderID:  "mockadex",
			Chapters: []models.ChapterResult{
				{Identifier: "ch1", Title: "Chapter 1"},
				{Identifier: "ch2", Title: "Chapter 2"},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/downloads/queue", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusAccepted {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
		}

		var count int
		server.db.QueryRow("SELECT COUNT(*) FROM download_queue").Scan(&count)
		if count != 2 {
			t.Errorf("Expected 2 items in queue, but found %d", count)
		}
	})
}

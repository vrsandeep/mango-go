package api

// A test file for the downloader API endpoints.
import (
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
}

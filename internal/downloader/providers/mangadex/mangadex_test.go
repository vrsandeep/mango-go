package mangadex

// It uses a mock HTTP server to avoid making real network requests.

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// setupTestServer creates a mock HTTP server to respond to API calls.
func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// Mock search endpoint
	mux.HandleFunc("/manga", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"series-1","attributes":{"title":{"en":"Test Manga"}},"relationships":[{"type":"cover_art","attributes":{"fileName":"cover.jpg"}}]}]}`)
	})

	// Mock chapter feed endpoint
	mux.HandleFunc("/manga/series-1/feed", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"id":"chapter-1","attributes":{"title":"Chapter One","volume":"1","chapter":"1","pages":20,"translatedLanguage":"en","publishAt":"2025-01-01T00:00:00Z"}}]}`)
	})

	// Mock page URL endpoint
	mux.HandleFunc("/at-home/server/chapter-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"baseUrl":"https://example.com","chapter":{"hash":"testhash","data":["page1.jpg","page2.jpg"]}}`)
	})

	return httptest.NewServer(mux)
}

func NewWithBaseURL(apiBaseURL string, coverArtUrl string) *MangaDexProvider {
	return &MangaDexProvider{
		client:          &http.Client{Timeout: 20 * time.Second},
		apiBaseURL:      apiBaseURL,
		coverArtBaseURL: coverArtUrl,
	}
}

func TestMangaDexProvider(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	// Create provider with mock server base URL
	p := NewWithBaseURL(server.URL, server.URL+"/coverArt")

	t.Run("Search", func(t *testing.T) {
		results, err := p.Search("test")
		if err != nil {
			t.Fatalf("Search() failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Expected 1 search result, got %d", len(results))
		}
		if results[0].Title != "Test Manga" {
			t.Errorf("Expected title 'Test Manga', got '%s'", results[0].Title)
		}
		if results[0].Identifier != "series-1" {
			t.Errorf("Expected identifier 'series-1', got '%s'", results[0].Identifier)
		}
	})

	t.Run("GetChapters", func(t *testing.T) {
		results, err := p.GetChapters("series-1")
		if err != nil {
			t.Fatalf("GetChapters() failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("Expected 1 chapter result, got %d", len(results))
		}
		if results[0].Title != "Vol. 1 Ch. 1 Chapter One" {
			t.Errorf("Expected title 'Vol. 1 Ch. 1 Chapter One', got '%s'", results[0].Title)
		}
		if results[0].Chapter != "1" {
			t.Errorf("Expected chapter '1', got '%s'", results[0].Chapter)
		}
	})

	t.Run("GetPageURLs", func(t *testing.T) {
		results, err := p.GetPageURLs("chapter-1")
		if err != nil {
			t.Fatalf("GetPageURLs() failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("Expected 2 page URLs, got %d", len(results))
		}
		expectedURL := "https://example.com/data/testhash/page1.jpg"
		if results[0] != expectedURL {
			t.Errorf("Expected URL '%s', got '%s'", expectedURL, results[0])
		}
	})
}

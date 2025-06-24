package weebcentral

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// Mock search endpoint (HTML response)
	mux.HandleFunc("/search/simple", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
		<div id="quick-search-result">
		  <div>
		    <a href="/series/series-1/">
		      <div class="flex-1">Test Manga</div>
		      <img src="https://example.com/cover.jpg" />
		    </a>
		  </div>
		</div>
		`)
	})

	// Mock chapters endpoint (HTML response)
	mux.HandleFunc("/series/series-1/full-chapter-list", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
		<div class="flex items-center">
		  <a href="/chapters/chapter-1">
		    <span class="grow">
		      <span>Vol. 1 Ch. 1 Chapter One</span>
		    </span>
		    <time datetime="2025-01-01T00:00:00Z"></time>
		  </a>
		</div>
		`)
	})

	// Mock pages endpoint (HTML response)
	mux.HandleFunc("/chapters/chapter-1/images", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `
		<section class="flex-1">
		  <img src="https://example.com/page1.jpg" />
		  <img src="https://example.com/page2.jpg" />
		</section>
		`)
	})

	return httptest.NewServer(mux)
}

func NewWithBaseURL(baseURL string) *WeebCentralProvider {
	return &WeebCentralProvider{
		client:  &http.Client{Timeout: 20 * time.Second},
		baseURL: baseURL,
	}
}

func TestWeebCentralProvider(t *testing.T) {
	server := setupTestServer()
	defer server.Close()

	p := NewWithBaseURL(server.URL)

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
		if !strings.HasPrefix(results[0].CoverURL, "https://example.com/cover.jpg") {
			t.Errorf("Expected cover url to start with 'https://example.com/cover.jpg', got '%s'", results[0].CoverURL)
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
		expectedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		if !results[0].PublishedAt.Equal(expectedTime) {
			t.Errorf("Expected PublishedAt %v, got %v", expectedTime, results[0].PublishedAt)
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
		expectedURL := "https://example.com/page1.jpg"
		if results[0] != expectedURL {
			t.Errorf("Expected URL '%s', got '%s'", expectedURL, results[0])
		}
	})
}

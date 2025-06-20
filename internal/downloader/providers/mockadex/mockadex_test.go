package mockadex

import (
	"strings"
	"testing"
)

func TestMockadexProvider(t *testing.T) {
	p := New()

	t.Run("GetInfo", func(t *testing.T) {
		info := p.GetInfo()
		if info.ID != "mockadex" || info.Name != "Mockadex" {
			t.Errorf("GetInfo() returned incorrect data: got %+v", info)
		}
	})

	t.Run("Search", func(t *testing.T) {
		results, err := p.Search("test query")
		if err != nil {
			t.Fatalf("Search() returned an error: %v", err)
		}
		if len(results) != 10 {
			t.Errorf("Search() expected 10 results, got %d", len(results))
		}
		if !strings.Contains(results[0].Title, "test query") {
			t.Errorf("Search() result title does not contain the query: %s", results[0].Title)
		}
	})

	t.Run("GetChapters", func(t *testing.T) {
		results, err := p.GetChapters("mock-series-1")
		if err != nil {
			t.Fatalf("GetChapters() returned an error: %v", err)
		}
		if len(results) != 25 {
			t.Errorf("GetChapters() expected 25 results, got %d", len(results))
		}
		if !strings.Contains(results[0].Identifier, "mock-series-1") {
			t.Errorf("GetChapters() result identifier is incorrect: %s", results[0].Identifier)
		}
	})

	t.Run("GetPageURLs", func(t *testing.T) {
		results, err := p.GetPageURLs("mock-chapter-1")
		if err != nil {
			t.Fatalf("GetPageURLs() returned an error: %v", err)
		}
		if len(results) != 20 {
			t.Errorf("GetPageURLs() expected 20 results, got %d", len(results))
		}
	})
}

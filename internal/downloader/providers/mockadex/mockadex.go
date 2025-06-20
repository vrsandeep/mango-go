// A mock provider for development and testing purposes. It simulates
// searching and fetching from a real site without making network calls.
package mockadex

import (
	"fmt"
	"github.com/vrsandeep/mango-go/internal/models"
	"strconv"
	"time"
)

type MockadexProvider struct{}

func New() *MockadexProvider {
	return &MockadexProvider{}
}

func (p *MockadexProvider) GetInfo() models.ProviderInfo {
	return models.ProviderInfo{
		ID:   "mockadex",
		Name: "Mockadex",
	}
}

func (p *MockadexProvider) Search(query string) ([]models.SearchResult, error) {
	var results []models.SearchResult
	for i := 1; i <= 10; i++ {
		results = append(results, models.SearchResult{
			Title:      fmt.Sprintf("%s - Result %d", query, i),
			CoverURL:   fmt.Sprintf("https://placehold.co/400x600/2a2a2a/f0f0f0?text=Cover+%d", i),
			Identifier: fmt.Sprintf("mock-series-%d", i),
		})
	}
	return results, nil
}

func (p *MockadexProvider) GetChapters(seriesIdentifier string) ([]models.ChapterResult, error) {
	var results []models.ChapterResult
	for i := 1; i <= 25; i++ {
		results = append(results, models.ChapterResult{
			Identifier:  fmt.Sprintf("mock-chapter-%s-%d", seriesIdentifier, i),
			Title:       fmt.Sprintf("Chapter %d: The Mocking", i),
			Volume:      "1",
			Chapter:     strconv.Itoa(i),
			Pages:       20 + i,
			Language:    "en",
			GroupID:     "mock-group",
			PublishedAt: time.Now().AddDate(0, 0, -i),
		})
	}
	return results, nil
}

func (p *MockadexProvider) GetPageURLs(chapterIdentifier string) ([]string, error) {
	var urls []string
	for i := 1; i <= 20; i++ {
		urls = append(urls, fmt.Sprintf("https://placehold.co/800x1200?text=Page+%d", i))
	}
	return urls, nil
}

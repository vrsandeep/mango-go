package mangadex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

// MangaDexProvider implements the Provider interface for MangaDex.
type MangaDexProvider struct {
	client          *http.Client
	apiBaseURL      string
	coverArtBaseURL string
}

// New creates a new instance of the MangaDexProvider.
func New() *MangaDexProvider {
	return &MangaDexProvider{
		client:          &http.Client{Timeout: 20 * time.Second},
		apiBaseURL:      "https://api.mangadex.org",
		coverArtBaseURL: "https://uploads.mangadex.org",
	}
}

// GetInfo returns static information about this provider.
func (p *MangaDexProvider) GetInfo() models.ProviderInfo {
	return models.ProviderInfo{
		ID:   "mangadex",
		Name: "MangaDex",
	}
}

// Search sends a request to the MangaDex API to search for manga.
func (p *MangaDexProvider) Search(query string) ([]models.SearchResult, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/manga", p.apiBaseURL), nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("title", query)
	q.Add("limit", "25")
	q.Add("includes[]", "cover_art")
	req.URL.RawQuery = q.Encode()

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResponse MangaListResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	var results []models.SearchResult
	for _, mangaData := range apiResponse.Data {
		title := mangaData.Attributes.Title.Get("en") // Default to English title
		if title == "" {
			// Fallback to first available title if English is not present
			for _, t := range mangaData.Attributes.Title {
				title = t
				break
			}
		}

		coverFileName := ""
		for _, rel := range mangaData.Relationships {
			if rel.Type == "cover_art" {
				coverFileName = rel.Attributes.FileName
				break
			}
		}

		coverURL := ""
		if coverFileName != "" {
			coverURL = fmt.Sprintf("%s/covers/%s/%s.256.jpg", p.coverArtBaseURL, mangaData.ID, coverFileName)
		}

		results = append(results, models.SearchResult{
			Title:      title,
			CoverURL:   coverURL,
			Identifier: mangaData.ID,
		})
	}

	return results, nil
}

// GetChapters fetches the chapter list for a given series from MangaDex.
func (p *MangaDexProvider) GetChapters(seriesIdentifier string) ([]models.ChapterResult, error) {
	var allChapters []models.ChapterResult
	offset := 0
	limit := 500

	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/manga/%s/feed", p.apiBaseURL, seriesIdentifier), nil)
		if err != nil {
			return nil, err
		}

		q := req.URL.Query()
		q.Add("limit", fmt.Sprintf("%d", limit))
		q.Add("offset", fmt.Sprintf("%d", offset))
		q.Add("order[volume]", "desc")
		q.Add("order[chapter]", "desc")
		// Filter for English chapters
		q.Add("translatedLanguage[]", "en")
		req.URL.RawQuery = q.Encode()

		resp, err := p.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var apiResponse ChapterFeedResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
			return nil, err
		}

		for _, chapterData := range apiResponse.Data {
			allChapters = append(allChapters, models.ChapterResult{
				Identifier:  chapterData.ID,
				Title:       chapterData.Attributes.Title,
				Volume:      chapterData.Attributes.Volume,
				Chapter:     chapterData.Attributes.Chapter,
				Pages:       chapterData.Attributes.Pages,
				Language:    chapterData.Attributes.TranslatedLanguage,
				PublishedAt: chapterData.Attributes.PublishAt,
			})
		}

		if len(apiResponse.Data) < limit {
			break // No more pages
		}
		offset += limit
	}

	// The API returns chapters in descending order, so we reverse to get ascending.
	sort.SliceStable(allChapters, func(i, j int) bool {
		return i > j
	})

	return allChapters, nil
}

// GetPageURLs retrieves the page URLs for a given chapter from MangaDex.
func (p *MangaDexProvider) GetPageURLs(chapterIdentifier string) ([]string, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/at-home/server/%s", p.apiBaseURL, chapterIdentifier), nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResponse AtHomeServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	baseURL := apiResponse.BaseURL
	hash := apiResponse.Chapter.Hash
	var pageURLs []string
	for _, pageFile := range apiResponse.Chapter.Data {
		pageURLs = append(pageURLs, fmt.Sprintf("%s/data/%s/%s", baseURL, hash, pageFile))
	}

	return pageURLs, nil
}

package plugins

import (
	"fmt"
	"log"
	"time"

	"github.com/dop251/goja"
	"github.com/vrsandeep/mango-go/internal/models"
)

// PluginProviderAdapter adapts a JavaScript plugin to the Provider interface.
type PluginProviderAdapter struct {
	runtime *PluginRuntime
}

// NewPluginProviderAdapter creates a new adapter for a plugin runtime.
func NewPluginProviderAdapter(runtime *PluginRuntime) *PluginProviderAdapter {
	return &PluginProviderAdapter{
		runtime: runtime,
	}
}

// GetInfo returns plugin information.
// Uses manifest directly
func (a *PluginProviderAdapter) GetInfo() models.ProviderInfo {
	name := a.runtime.manifest.Name
	if name == "" {
		// If name is not in manifest, use ID as fallback
		name = a.runtime.manifest.ID
	}

	return models.ProviderInfo{
		ID:   a.runtime.manifest.ID,
		Name: name,
	}
}

// Search searches for series.
func (a *PluginProviderAdapter) Search(query string) ([]models.SearchResult, error) {
	val, err := a.runtime.Call("search", query)
	if err != nil {
		return nil, &PluginError{
			PluginID: a.runtime.manifest.ID,
			Function: "search",
			Message:  fmt.Sprintf("search failed: %v", err),
			Cause:    err,
		}
	}

	// Check if value is valid before conversion
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return nil, &PluginError{
			PluginID: a.runtime.manifest.ID,
			Function: "search",
			Message:  "search returned invalid value (null or undefined)",
		}
	}

	results := a.jsToSearchResults(val)
	if results == nil {
		// Return empty array instead of nil for consistency
		return []models.SearchResult{}, nil
	}

	return results, nil
}

// GetChapters gets chapters for a series.
func (a *PluginProviderAdapter) GetChapters(seriesIdentifier string) ([]models.ChapterResult, error) {
	val, err := a.runtime.Call("getChapters", seriesIdentifier)
	if err != nil {
		return nil, &PluginError{
			PluginID: a.runtime.manifest.ID,
			Function: "getChapters",
			Message:  fmt.Sprintf("getChapters failed: %v", err),
			Cause:    err,
		}
	}

	chapters := a.jsToChapterResults(val)
	return chapters, nil
}

// GetPageURLs gets page URLs for a chapter.
func (a *PluginProviderAdapter) GetPageURLs(chapterIdentifier string) ([]string, error) {
	val, err := a.runtime.Call("getPageURLs", chapterIdentifier)
	if err != nil {
		return nil, &PluginError{
			PluginID: a.runtime.manifest.ID,
			Function: "getPageURLs",
			Message:  fmt.Sprintf("getPageURLs failed: %v", err),
			Cause:    err,
		}
	}

	urls := a.jsToStringArray(val)
	return urls, nil
}

// Helper functions to convert JS values to Go types

func (a *PluginProviderAdapter) jsToSearchResults(val goja.Value) []models.SearchResult {
	// Use named return to allow recover to modify return value
	var results []models.SearchResult

	// Recover from any panics and return empty slice
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[%s] Panic in jsToSearchResults: %v", a.runtime.manifest.ID, r)
			results = []models.SearchResult{}
		}
	}()

	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return []models.SearchResult{}
	}

	if a.runtime == nil || a.runtime.vm == nil {
		log.Printf("[plugin] Runtime or VM is nil in jsToSearchResults")
		return []models.SearchResult{}
	}

	// Try to export the value directly - goja can convert JS arrays to []interface{}
	exported := val.Export()
	if exported == nil {
		log.Printf("[%s] Failed to export search result value", a.runtime.manifest.ID)
		return []models.SearchResult{}
	}

	// Check if it's a slice/array
	arr, ok := exported.([]interface{})
	if !ok {
		log.Printf("[%s] Search result is not an array, got type: %T", a.runtime.manifest.ID, exported)
		return []models.SearchResult{}
	}

	if len(arr) == 0 {
		return []models.SearchResult{}
	}

	results = make([]models.SearchResult, 0, len(arr))
	for _, itemInterface := range arr {
		itemMap, ok := itemInterface.(map[string]interface{})
		if !ok {
			log.Printf("[%s] Search result item is not an object, got type: %T", a.runtime.manifest.ID, itemInterface)
			continue
		}

		result := models.SearchResult{
			Title:      a.getStringFromMap(itemMap, "title"),
			CoverURL:   a.getStringFromMap(itemMap, "cover_url"),
			Identifier: a.getStringFromMap(itemMap, "identifier"),
		}
		results = append(results, result)
	}

	return results
}

func (a *PluginProviderAdapter) getStringFromMap(m map[string]interface{}, key string) string {
	val, ok := m[key]
	if !ok || val == nil {
		return ""
	}
	if str, ok := val.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", val)
}

func (a *PluginProviderAdapter) jsToChapterResults(val goja.Value) []models.ChapterResult {
	if goja.IsUndefined(val) || goja.IsNull(val) {
		return nil
	}

	// Export the value and work with Go types
	exported := val.Export()
	if exported == nil {
		return nil
	}

	arr, ok := exported.([]interface{})
	if !ok {
		return nil
	}

	if len(arr) == 0 {
		return []models.ChapterResult{}
	}

	chapters := make([]models.ChapterResult, 0, len(arr))
	for _, itemInterface := range arr {
		itemMap, ok := itemInterface.(map[string]interface{})
		if !ok {
			continue
		}

		// Parse published_at
		var publishedAt time.Time
		if publishedAtStr := a.getStringFromMap(itemMap, "published_at"); publishedAtStr != "" {
			if parsed, err := time.Parse(time.RFC3339, publishedAtStr); err == nil {
				publishedAt = parsed
			} else if parsed, err := time.Parse("2006-01-02T15:04:05Z", publishedAtStr); err == nil {
				publishedAt = parsed
			}
		}

		// Get pages as number
		pages := 0
		if pagesVal, ok := itemMap["pages"]; ok && pagesVal != nil {
			if pagesFloat, ok := pagesVal.(float64); ok {
				pages = int(pagesFloat)
			} else if pagesInt, ok := pagesVal.(int); ok {
				pages = pagesInt
			}
		}

		chapter := models.ChapterResult{
			Identifier:  a.getStringFromMap(itemMap, "identifier"),
			Title:       a.getStringFromMap(itemMap, "title"),
			Volume:      a.getStringFromMap(itemMap, "volume"),
			Chapter:     a.getStringFromMap(itemMap, "chapter"),
			Pages:       pages,
			Language:    a.getStringFromMap(itemMap, "language"),
			GroupID:     a.getStringFromMap(itemMap, "group_id"),
			PublishedAt: publishedAt,
		}
		chapters = append(chapters, chapter)
	}

	return chapters
}

func (a *PluginProviderAdapter) jsToStringArray(val goja.Value) []string {
	if goja.IsUndefined(val) || goja.IsNull(val) {
		return nil
	}

	// Export the value and work with Go types
	exported := val.Export()
	if exported == nil {
		return nil
	}

	arr, ok := exported.([]interface{})
	if !ok {
		return nil
	}

	if len(arr) == 0 {
		return []string{}
	}

	urls := make([]string, 0, len(arr))
	for _, itemInterface := range arr {
		if itemInterface == nil {
			continue
		}
		if str, ok := itemInterface.(string); ok {
			urls = append(urls, str)
		} else {
			urls = append(urls, fmt.Sprintf("%v", itemInterface))
		}
	}

	return urls
}

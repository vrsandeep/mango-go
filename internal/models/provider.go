package models

import "time"

// ProviderInfo contains static information about a provider.
type ProviderInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SearchResult represents a single series found by a provider.
type SearchResult struct {
	Title      string `json:"title"`
	CoverURL   string `json:"cover_url"`
	Identifier string `json:"identifier"` // Unique ID for the series on the source site
}

// ChapterResult represents a single chapter for a series from a provider.
type ChapterResult struct {
	Identifier  string    `json:"identifier"` // Unique ID for the chapter on the source site
	Title       string    `json:"title"`
	Volume      string    `json:"volume"`
	Chapter     string    `json:"chapter"`
	Pages       int       `json:"pages"`
	Language    string    `json:"language"`
	GroupID     string    `json:"group_id"`
	PublishedAt time.Time `json:"published_at"`
}

// Provider defines the contract that every website connector must implement.
type Provider interface {
	GetInfo() ProviderInfo
	Search(query string) ([]SearchResult, error)
	GetChapters(seriesIdentifier string) ([]ChapterResult, error)
	GetPageURLs(chapterIdentifier string) ([]string, error)
}

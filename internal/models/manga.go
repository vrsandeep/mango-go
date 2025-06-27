// This file defines the core data structures (models) for our application.
// These structs represent the manga, chapters, and pages in our library.

package models

import "time"

// User represents a user account.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose password hash
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
}

// Series represents a single manga series.
type Series struct {
	ID             int64      `json:"id"`
	Title          string     `json:"title"`
	Path           string     `json:"path"`
	Thumbnail      string     `json:"thumbnail,omitempty"`
	CustomCoverURL string     `json:"custom_cover_url,omitempty"` // New field
	Chapters       []*Chapter `json:"chapters,omitempty"`         // omitempty hides it when not loaded
	Tags           []*Tag     `json:"tags,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Fields calculated by the API, not stored in DB
	TotalChapters int             `json:"total_chapters,omitempty"`
	ReadChapters  int             `json:"read_chapters,omitempty"`
	Settings      *SeriesSettings `json:"settings,omitempty"` // Series-specific settings
}

// Chapter represents a single chapter of a manga.
type Chapter struct {
	ID              int64     `json:"id"`
	SeriesID        int64     `json:"series_id"`
	Path            string    `json:"path"`
	Thumbnail       string    `json:"thumbnail,omitempty"`
	PageCount       int       `json:"page_count"`
	Read            bool      `json:"read"`
	ProgressPercent int       `json:"progress_percent"`
	CreatedAt       time.Time `json:"-"` // Hide from JSON responses
	UpdatedAt       time.Time `json:"-"` // Hide from JSON responses
}

// Page represents a single page within a chapter, which is an image
// file inside the archive.
type Page struct {
	FileName string `json:"file_name"`
	Index    int    `json:"index"`
}

type Tag struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	SeriesCount int    `json:"series_count,omitempty"`
}

// SeriesSettings holds per-user sort preferences for a series.
type SeriesSettings struct {
	SortBy   string `json:"sort_by"`  // e.g. "auto", "path"
	SortDir  string `json:"sort_dir"` // e.g. "asc", "desc"
	SeriesID int64  `json:"-"`        // Hide from JSON responses
	// UserID   int64  `json:"-"`        // Hide from JSON responses
}

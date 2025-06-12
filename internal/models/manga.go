// This file defines the core data structures (models) for our application.
// These structs represent the manga, chapters, and pages in our library.

package models

import "time"

// Series represents a single manga series.
type Series struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Path      string     `json:"path"`
	Chapters  []*Chapter `json:"chapters,omitempty"` // omitempty hides it when not loaded
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Chapter represents a single chapter of a manga.
type Chapter struct {
	ID              int64     `json:"id"`
	SeriesID        int64     `json:"series_id"`
	Path            string    `json:"path"`
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

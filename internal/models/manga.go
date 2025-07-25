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

// Folder represents a directory in the user's library.
type Folder struct {
	ID        int64     `json:"id"`
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	ParentID  *int64    `json:"parent_id"`
	Thumbnail string    `json:"thumbnail,omitempty"`
	Tags      []*Tag    `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// For API responses
	Subfolders []*Folder  `json:"subfolders,omitempty"`
	Chapters   []*Chapter `json:"chapters,omitempty"`

	// Fields calculated by the API, not stored in DB
	TotalChapters int             `json:"total_chapters,omitempty"`
	ReadChapters  int             `json:"read_chapters,omitempty"`
	// Settings      *FolderSettings `json:"settings,omitempty"` // Folder-specific settings
}

// Chapter represents a single chapter of a manga.
type Chapter struct {
	ID          int64     `json:"id"`
	FolderID    int64     `json:"folder_id"`
	Path        string    `json:"path"`
	ContentHash string    `json:"-"` // Internal use, not exposed in API
	Thumbnail   string    `json:"thumbnail,omitempty"`
	PageCount   int       `json:"page_count"`
	CreatedAt   time.Time `json:"created_at"` // `json:"-"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Per-user progress
	Read            bool `json:"read"`
	ProgressPercent int  `json:"progress_percent"`
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
	FolderCount int    `json:"folder_count,omitempty"`
}

type FolderSettings struct {
	SortBy   string `json:"sort_by"`  // e.g. "auto", "path"
	SortDir  string `json:"sort_dir"` // e.g. "asc", "desc"
	FolderID int64  `json:"-"`        // Hide from JSON responses
}

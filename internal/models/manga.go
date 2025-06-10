// This file defines the core data structures (models) for our application.
// These structs represent the manga, chapters, and pages in our library.

package models

// Manga represents a single manga series. It contains multiple chapters.
type Manga struct {
	Title    string     `json:"title"`
	Path     string     `json:"path"` // The directory path for the series
	Chapters []*Chapter `json:"chapters"`
}

// Chapter represents a single chapter of a manga. It corresponds to one
// archive file (e.g., .cbz) and contains multiple pages.
type Chapter struct {
	FileName  string  `json:"file_name"`
	Path      string  `json:"path"` // The full path to the archive file
	PageCount int     `json:"page_count"`
	Pages     []*Page `json:"-"` // We hide pages from top-level JSON for brevity
}

// Page represents a single page within a chapter, which is an image
// file inside the archive.
type Page struct {
	FileName string `json:"file_name"`
	Index    int    `json:"index"`
}

package models

import "time"

// HomeSectionItem is a generic struct representing a card on the home page.
// It can represent a chapter or a series.
type HomeSectionItem struct {
	SeriesID        int64     `json:"series_id"`
	SeriesTitle     string    `json:"series_title"`
	ChapterID       *int64    `json:"chapter_id,omitempty"`
	ChapterTitle    string    `json:"chapter_title,omitempty"`
	CoverArt        string    `json:"cover_art"`
	ProgressPercent *int      `json:"progress_percent,omitempty"`
	Read            *bool     `json:"read,omitempty"`
	UpdatedAt       time.Time `json:"-"` // Used for sorting
	NewChapterCount int       `json:"new_chapter_count,omitempty"`
}

// HomePageData is the top-level struct for the /api/home response.
type HomePageData struct {
	ContinueReading []*HomeSectionItem `json:"continue_reading"`
	NextUp          []*HomeSectionItem `json:"next_up"`
	RecentlyAdded   []*HomeSectionItem `json:"recently_added"`
	StartReading    []*HomeSectionItem `json:"start_reading"`
}

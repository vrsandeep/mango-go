package models

import "time"

type Subscription struct {
	ID               int64      `json:"id"`
	SeriesTitle      string     `json:"series_title"`
	SeriesIdentifier string     `json:"series_identifier"`
	ProviderID       string     `json:"provider_id"`
	LocalSeriesID    *int64     `json:"local_series_id,omitempty"` // Nullable, links to local library if matched
	LastCheckedAt    *time.Time `json:"last_checked_at,omitempty"` // Nullable, when the series was last checked for updates
	CreatedAt        time.Time  `json:"created_at"`
}

type DownloadQueueItem struct {
	ID                int64     `json:"id"`
	SeriesTitle       string    `json:"series_title"`
	ChapterTitle      string    `json:"chapter_title"`
	ChapterIdentifier string    `json:"chapter_identifier"`
	Status            string    `json:"status"`   // e.g. "pending", "downloading", "completed", "failed"
	Progress          int       `json:"progress"` // Percentage of download progress
	Message           string    `json:"message"`  // Optional message for status updates
	ProviderID        string    `json:"provider_id"`
	CreatedAt         time.Time `json:"created_at"`
}

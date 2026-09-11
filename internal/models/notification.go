package models

import "time"

const NotificationKindChapterDownloaded = "chapter_downloaded"

// Notification is a library event shown in the header bell, such as a newly downloaded chapter.
type Notification struct {
	ID             int64     `json:"id"`
	Kind           string    `json:"kind"`
	SeriesTitle    string    `json:"series_title"`
	ChapterTitle   string    `json:"chapter_title"`
	LocalFolderID  *int64    `json:"local_folder_id,omitempty"`
	LocalChapterID *int64    `json:"local_chapter_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	Read           bool      `json:"read"`
}

// NotificationList is the payload for the notifications API.
type NotificationList struct {
	HasUnread     bool            `json:"has_unread"`
	HasMore       bool            `json:"has_more"`
	Notifications []*Notification `json:"notifications"`
}

package mangadex

import "time"

// --- Common Types ---
type Relationship struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Attributes struct {
		FileName string `json:"fileName"`
	} `json:"attributes"`
}

type MultiLingualString map[string]string

func (mls MultiLingualString) Get(lang string) string {
	if val, ok := mls[lang]; ok {
		return val
	}
	return ""
}

// --- Manga Search Types ---
type MangaListResponse struct {
	Data []MangaData `json:"data"`
}
type MangaData struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Attributes    MangaAttributes `json:"attributes"`
	Relationships []Relationship  `json:"relationships"`
}
type MangaAttributes struct {
	Title MultiLingualString `json:"title"`
}

// --- Chapter Feed Types ---
type ChapterFeedResponse struct {
	Data  []ChapterData `json:"data"`
	Limit int           `json:"limit"`
	Total int           `json:"total"`
}
type ChapterData struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Attributes ChapterAttributes `json:"attributes"`
}
type ChapterAttributes struct {
	Title              string    `json:"title"`
	Volume             string    `json:"volume"`
	Chapter            string    `json:"chapter"`
	Pages              int       `json:"pages"`
	TranslatedLanguage string    `json:"translatedLanguage"`
	PublishAt          time.Time `json:"publishAt"`
}

// --- Page URL Types ---
type AtHomeServerResponse struct {
	BaseURL string `json:"baseUrl"`
	Chapter struct {
		Hash string   `json:"hash"`
		Data []string `json:"data"` // Page filenames
	} `json:"chapter"`
}

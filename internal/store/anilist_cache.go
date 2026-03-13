package store

import (
	"database/sql"
	"errors"
	"time"
)

var ErrAnilistCacheNotFound = errors.New("anilist cache not found")

// FolderAnilistCache represents a cached AniList Media result for a folder.
type FolderAnilistCache struct {
	FolderID      int64
	AnilistID     int64
	SiteURL       string
	CoverImageURL string
	TitleRomaji   string
	TitleEnglish  string
	CreatedAt     time.Time
}

// GetFolderAnilist returns cached AniList data for the folder, or ErrAnilistCacheNotFound.
func (s *Store) GetFolderAnilist(folderID int64) (*FolderAnilistCache, error) {
	query := `SELECT folder_id, anilist_id, site_url, cover_image_url, title_romaji, title_english, created_at
	          FROM folder_anilist_cache WHERE folder_id = ?`
	var c FolderAnilistCache
	var coverURL, titleRomaji, titleEnglish sql.NullString
	err := s.db.QueryRow(query, folderID).Scan(
		&c.FolderID, &c.AnilistID, &c.SiteURL, &coverURL, &titleRomaji, &titleEnglish, &c.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrAnilistCacheNotFound
		}
		return nil, err
	}
	if coverURL.Valid {
		c.CoverImageURL = coverURL.String
	}
	if titleRomaji.Valid {
		c.TitleRomaji = titleRomaji.String
	}
	if titleEnglish.Valid {
		c.TitleEnglish = titleEnglish.String
	}
	return &c, nil
}

// SetFolderAnilist inserts or replaces AniList cache for the folder.
func (s *Store) SetFolderAnilist(folderID int64, anilistID int64, siteURL, coverImageURL, titleRomaji, titleEnglish string) error {
	query := `INSERT INTO folder_anilist_cache (folder_id, anilist_id, site_url, cover_image_url, title_romaji, title_english, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(folder_id) DO UPDATE SET
	            anilist_id = excluded.anilist_id,
	            site_url = excluded.site_url,
	            cover_image_url = excluded.cover_image_url,
	            title_romaji = excluded.title_romaji,
	            title_english = excluded.title_english,
	            created_at = excluded.created_at`
	var cover, romaji, english interface{}
	if coverImageURL != "" {
		cover = coverImageURL
	}
	if titleRomaji != "" {
		romaji = titleRomaji
	}
	if titleEnglish != "" {
		english = titleEnglish
	}
	_, err := s.db.Exec(query, folderID, anilistID, siteURL, cover, romaji, english, time.Now())
	return err
}

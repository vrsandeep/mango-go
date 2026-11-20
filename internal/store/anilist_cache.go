package store

import (
	"database/sql"
	"errors"
	"time"
)

var ErrAnilistCacheNotFound = errors.New("anilist cache not found")

// FolderAnilistCache represents a cached AniList Media result for a folder.
// BannerFetched is false when banner_path is NULL (not downloaded yet).
// An empty BannerPath with BannerFetched true means AniList has no banner.
// BannerPath is relative to the data directory, e.g. banners/series-banner-ab12c.jpg.
type FolderAnilistCache struct {
	FolderID      int64
	AnilistID     int64
	SiteURL       string
	CoverImageURL string
	BannerPath    string
	BannerFetched bool
	TitleRomaji   string
	TitleEnglish  string
	CreatedAt     time.Time
}

// GetFolderAnilist returns cached AniList data for the folder, or ErrAnilistCacheNotFound.
func (s *Store) GetFolderAnilist(folderID int64) (*FolderAnilistCache, error) {
	query := `SELECT folder_id, anilist_id, site_url, cover_image_url, banner_path, title_romaji, title_english, created_at
	          FROM folder_anilist_cache WHERE folder_id = ?`
	var c FolderAnilistCache
	var coverURL, bannerURL, titleRomaji, titleEnglish sql.NullString
	err := s.db.QueryRow(query, folderID).Scan(
		&c.FolderID, &c.AnilistID, &c.SiteURL, &coverURL, &bannerURL, &titleRomaji, &titleEnglish, &c.CreatedAt,
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
	if bannerURL.Valid {
		c.BannerPath = bannerURL.String
		c.BannerFetched = true
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
// SetFolderAnilist inserts or replaces AniList cache for the folder.
// bannerKnown false writes NULL so a later visit retries the download.
// bannerKnown true stores bannerPath, including "" when AniList has no banner.
func (s *Store) SetFolderAnilist(folderID int64, anilistID int64, siteURL, coverImageURL, titleRomaji, titleEnglish, bannerPath string, bannerKnown bool) error {
	query := `INSERT INTO folder_anilist_cache (folder_id, anilist_id, site_url, cover_image_url, banner_path, title_romaji, title_english, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(folder_id) DO UPDATE SET
	            anilist_id = excluded.anilist_id,
	            site_url = excluded.site_url,
	            cover_image_url = excluded.cover_image_url,
	            banner_path = excluded.banner_path,
	            title_romaji = excluded.title_romaji,
	            title_english = excluded.title_english,
	            created_at = excluded.created_at`
	var cover, romaji, english, banner interface{}
	if coverImageURL != "" {
		cover = coverImageURL
	}
	if titleRomaji != "" {
		romaji = titleRomaji
	}
	if titleEnglish != "" {
		english = titleEnglish
	}
	if bannerKnown {
		banner = bannerPath
	}
	_, err := s.db.Exec(query, folderID, anilistID, siteURL, cover, banner, romaji, english, time.Now())
	return err
}

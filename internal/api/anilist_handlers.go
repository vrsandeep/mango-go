package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/anilist"
	"github.com/vrsandeep/mango-go/internal/banners"
	"github.com/vrsandeep/mango-go/internal/store"
)

// AnilistSearcher fetches manga metadata from AniList. Tests inject a fake.
type AnilistSearcher interface {
	SearchManga(title string) (*anilist.Media, error)
}

// SetAnilistSearcher replaces the AniList client used by the folder handlers.
func (s *Server) SetAnilistSearcher(searcher AnilistSearcher) {
	s.anilistSearcher = searcher
}

func (s *Server) searchAnilist(name string) (*anilist.Media, error) {
	if s.anilistSearcher != nil {
		return s.anilistSearcher.SearchManga(name)
	}
	return anilist.SearchManga(name)
}

// anilistMediaResponse is the JSON shape returned to the frontend (matches AniList Media).
type anilistMediaResponse struct {
	ID          int64              `json:"id"`
	SiteURL     string             `json:"siteUrl"`
	Title       *anilistTitle      `json:"title"`
	CoverImage  *anilistCoverImage `json:"coverImage"`
	BannerImage string             `json:"bannerImage"`
}

type anilistTitle struct {
	Romaji  string `json:"romaji"`
	English string `json:"english"`
}

type anilistCoverImage struct {
	Large string `json:"large"`
}

func anilistResponseFromCache(c *store.FolderAnilistCache) anilistMediaResponse {
	var title *anilistTitle
	if c.TitleRomaji != "" || c.TitleEnglish != "" {
		title = &anilistTitle{Romaji: c.TitleRomaji, English: c.TitleEnglish}
	}
	var cover *anilistCoverImage
	if c.CoverImageURL != "" {
		cover = &anilistCoverImage{Large: c.CoverImageURL}
	}
	return anilistMediaResponse{
		ID:          c.AnilistID,
		SiteURL:     c.SiteURL,
		Title:       title,
		CoverImage:  cover,
		BannerImage: banners.PublicURL(c.BannerPath),
	}
}

// refreshUnfetchedBanner re-queries AniList when banner_path is NULL.
// A download or network error leaves NULL so the next visit can retry.
// A successful lookup stores a local file path, or "" when AniList has no banner.
func (s *Server) refreshUnfetchedBanner(folderID int64, folderName string, cached *store.FolderAnilistCache) *store.FolderAnilistCache {
	if cached.BannerFetched {
		return cached
	}
	media, err := s.searchAnilist(folderName)
	if err != nil {
		log.Printf("AniList banner refresh error: %v", err)
		return cached
	}

	anilistID := cached.AnilistID
	siteURL := cached.SiteURL
	coverURL := cached.CoverImageURL
	titleRomaji := cached.TitleRomaji
	titleEnglish := cached.TitleEnglish
	bannerPath := ""
	bannerKnown := true
	if media != nil && media.SiteURL != "" {
		anilistID = media.ID
		siteURL = media.SiteURL
		if media.Title != nil {
			titleRomaji = media.Title.Romaji
			titleEnglish = media.Title.English
		}
		if media.CoverImage != nil {
			coverURL = media.CoverImage.Large
		}
		bannerPath, bannerKnown = s.localBanner(folderName, media.BannerImage)
	}
	if err := s.store.SetFolderAnilist(folderID, anilistID, siteURL, coverURL, titleRomaji, titleEnglish, bannerPath, bannerKnown); err != nil {
		log.Printf("SetFolderAnilist banner refresh: %v", err)
		return cached
	}
	updated, err := s.store.GetFolderAnilist(folderID)
	if err != nil {
		log.Printf("GetFolderAnilist after banner refresh: %v", err)
		return cached
	}
	return updated
}

// handleGetFolderAnilist returns cached AniList data for the folder, or 404 if not cached.
func (s *Server) handleGetFolderAnilist(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}
	folder, err := s.store.GetFolder(folderID)
	if err != nil {
		if err == store.ErrFolderNotFound {
			RespondWithError(w, http.StatusNotFound, "Folder not found")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to load folder")
		return
	}
	cached, err := s.store.GetFolderAnilist(folderID)
	if err != nil {
		if err == store.ErrAnilistCacheNotFound {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		log.Printf("GetFolderAnilist: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to load AniList cache")
		return
	}
	if !cached.BannerFetched {
		cached = s.refreshUnfetchedBanner(folderID, folder.Name, cached)
	}
	RespondWithJSON(w, http.StatusOK, anilistResponseFromCache(cached))
}

// handlePostFolderAnilist fetches from AniList using the folder name, stores in DB, and returns the data.
func (s *Server) handlePostFolderAnilist(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}
	folder, err := s.store.GetFolder(folderID)
	if err != nil {
		if err == store.ErrFolderNotFound {
			RespondWithError(w, http.StatusNotFound, "Folder not found")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to load folder")
		return
	}
	media, err := s.searchAnilist(folder.Name)
	if err != nil {
		log.Printf("AniList SearchManga error: %v", err)
		RespondWithError(w, http.StatusBadGateway, "Failed to fetch from AniList")
		return
	}
	if media == nil || media.SiteURL == "" {
		// Remember the miss so the next visit does not call AniList again.
		if err := s.store.SetFolderAnilist(folderID, 0, "", "", "", "", "", true); err != nil {
			log.Printf("SetFolderAnilist miss: %v", err)
			RespondWithError(w, http.StatusInternalServerError, "Failed to save AniList cache")
			return
		}
		RespondWithError(w, http.StatusNotFound, "No AniList manga found for this folder")
		return
	}
	titleRomaji, titleEnglish := "", ""
	if media.Title != nil {
		titleRomaji = media.Title.Romaji
		titleEnglish = media.Title.English
	}
	coverURL := ""
	if media.CoverImage != nil {
		coverURL = media.CoverImage.Large
	}
	bannerPath, bannerKnown := s.localBanner(folder.Name, media.BannerImage)
	if err := s.store.SetFolderAnilist(folderID, media.ID, media.SiteURL, coverURL, titleRomaji, titleEnglish, bannerPath, bannerKnown); err != nil {
		log.Printf("SetFolderAnilist: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to save AniList cache")
		return
	}
	resp := anilistMediaResponse{
		ID:          media.ID,
		SiteURL:     media.SiteURL,
		Title:       nil,
		CoverImage:  nil,
		BannerImage: banners.PublicURL(bannerPath),
	}
	if media.Title != nil {
		resp.Title = &anilistTitle{Romaji: media.Title.Romaji, English: media.Title.English}
	}
	if media.CoverImage != nil && media.CoverImage.Large != "" {
		resp.CoverImage = &anilistCoverImage{Large: media.CoverImage.Large}
	}
	RespondWithJSON(w, http.StatusOK, resp)
}

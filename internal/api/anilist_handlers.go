package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/anilist"
	"github.com/vrsandeep/mango-go/internal/store"
)

// anilistMediaResponse is the JSON shape returned to the frontend (matches AniList Media).
type anilistMediaResponse struct {
	ID         int64              `json:"id"`
	SiteURL    string             `json:"siteUrl"`
	Title      *anilistTitle      `json:"title"`
	CoverImage *anilistCoverImage `json:"coverImage"`
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
		ID:         c.AnilistID,
		SiteURL:    c.SiteURL,
		Title:      title,
		CoverImage: cover,
	}
}

// handleGetFolderAnilist returns cached AniList data for the folder, or 404 if not cached.
func (s *Server) handleGetFolderAnilist(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil || folderID <= 0 {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}
	if _, err := s.store.GetFolder(folderID); err != nil {
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
	media, err := anilist.SearchManga(folder.Name)
	if err != nil {
		log.Printf("AniList SearchManga error: %v", err)
		RespondWithError(w, http.StatusBadGateway, "Failed to fetch from AniList")
		return
	}
	if media == nil || media.SiteURL == "" {
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
	if err := s.store.SetFolderAnilist(folderID, media.ID, media.SiteURL, coverURL, titleRomaji, titleEnglish); err != nil {
		log.Printf("SetFolderAnilist: %v", err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to save AniList cache")
		return
	}
	resp := anilistMediaResponse{
		ID:         media.ID,
		SiteURL:    media.SiteURL,
		Title:      nil,
		CoverImage: nil,
	}
	if media.Title != nil {
		resp.Title = &anilistTitle{Romaji: media.Title.Romaji, English: media.Title.English}
	}
	if media.CoverImage != nil && media.CoverImage.Large != "" {
		resp.CoverImage = &anilistCoverImage{Large: media.CoverImage.Large}
	}
	RespondWithJSON(w, http.StatusOK, resp)
}

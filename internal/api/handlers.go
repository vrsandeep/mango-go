package api

import (
	"archive/zip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/models"
)

// getListParams extracts all query params for list endpoints.
func getListParams(r *http.Request) (page, perPage int, search, sortBy, sortDir string) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ = strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage <= 0 || perPage > 100 { // Enforce a max of 100
		perPage = 100
	}
	search = r.URL.Query().Get("search")
	sortBy = r.URL.Query().Get("sort_by")
	sortDir = r.URL.Query().Get("sort_dir")
	return
}

// handleListSeries is updated to handle search and sort.
func (s *Server) handleListSeries(w http.ResponseWriter, r *http.Request) {
	page, perPage, search, sortBy, sortDir := getListParams(r)

	series, total, err := s.store.ListSeries(page, perPage, search, sortBy, sortDir)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve series from database")
		return
	}
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	RespondWithJSON(w, http.StatusOK, series)
}

// handleGetSeries retrieves and returns details for a single manga series.
func (s *Server) handleGetSeries(w http.ResponseWriter, r *http.Request) {
	seriesIDStr := chi.URLParam(r, "seriesID")
	seriesID, err := strconv.ParseInt(seriesIDStr, 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid series ID")
		return
	}

	settings, err := s.store.GetSeriesSettings(seriesID)
	if err != nil {
		log.Printf("Failed to retrieve settings for series %d: %v", seriesID, err)
	}
	page, perPage, search, sortBy, sortDir := getListParams(r)

	// save settings to series if they exist
	if sortBy != "" || sortDir != "" {
		s.store.UpdateSeriesSettings(seriesID, sortBy, sortDir)
	} else {
		sortBy = settings.SortBy   // Use default sort from settings if not specified
		sortDir = settings.SortDir // Use default direction from settings if not specified
	}

	series, total, err := s.store.GetSeriesByID(seriesID, page, perPage, search, sortBy, sortDir)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "Series not found")
		return
	}

	series.Settings = &models.SeriesSettings{
		SortBy:  sortBy,
		SortDir: sortDir,
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	RespondWithJSON(w, http.StatusOK, series)
}

// handleUpdateCover handles requests to update the custom cover URL for a series.
func (s *Server) handleUpdateCover(w http.ResponseWriter, r *http.Request) {
	seriesIDStr := chi.URLParam(r, "seriesID")
	seriesID, err := strconv.ParseInt(seriesIDStr, 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid series ID")
		return
	}

	var payload struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if payload.URL == "" {
		RespondWithError(w, http.StatusBadRequest, "Cover URL cannot be empty")
		return
	}

	if rowsAffected, err := s.store.UpdateSeriesCoverURL(seriesID, payload.URL); err != nil {
		log.Printf("Failed to update cover for series %d: %v", seriesID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to update cover")
		return
	} else if rowsAffected == 0 {
		RespondWithError(w, http.StatusNotFound, "Series not found")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleMarkAllAs marks all chapters in a series as read or unread.
func (s *Server) handleMarkAllAs(w http.ResponseWriter, r *http.Request) {
	seriesIDStr := chi.URLParam(r, "seriesID")
	seriesID, err := strconv.ParseInt(seriesIDStr, 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid series ID")
		return
	}

	var payload struct {
		Read bool `json:"read"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := s.store.MarkAllChaptersAs(seriesID, payload.Read); err != nil {
		log.Printf("Failed to mark all chapters for series %d: %v", seriesID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to update chapters")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleAddTag(w http.ResponseWriter, r *http.Request) {
	seriesID, _ := strconv.ParseInt(chi.URLParam(r, "seriesID"), 10, 64)
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	tag, err := s.store.AddTagToSeries(seriesID, payload.Name)
	if err != nil {
		log.Printf("Failed to add tag to series %d: %v", seriesID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to add tag")
		return
	}
	RespondWithJSON(w, http.StatusCreated, tag)
}

func (s *Server) handleRemoveTag(w http.ResponseWriter, r *http.Request) {
	seriesID, _ := strconv.ParseInt(chi.URLParam(r, "seriesID"), 10, 64)
	tagID, _ := strconv.ParseInt(chi.URLParam(r, "tagID"), 10, 64)
	if err := s.store.RemoveTagFromSeries(seriesID, tagID); err != nil {
		log.Printf("Failed to remove tag %d from series %d: %v", tagID, seriesID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to remove tag")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGetPage finds a specific page within an archive and serves it as an image.
func (s *Server) handleGetPage(w http.ResponseWriter, r *http.Request) {
	chapterIDStr := chi.URLParam(r, "chapterID")
	chapterID, err := strconv.ParseInt(chapterIDStr, 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid chapter ID")
		return
	}

	pageNumberStr := chi.URLParam(r, "pageNumber")
	// Page numbers are 1-based for the user, convert to 0-based index
	pageNumber, err := strconv.Atoi(pageNumberStr)
	if err != nil || pageNumber < 1 {
		RespondWithError(w, http.StatusBadRequest, "Invalid page number")
		return
	}
	pageIndex := pageNumber - 1

	// Get chapter details (we need its path) from the database
	chapter, err := s.store.GetChapterByID(chapterID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "Chapter not found")
		return
	}

	// --- Image Extraction Logic ---
	// For now, we only support .cbz (zip) files.
	// Future: This could be expanded to dispatch to a CBR parser.
	if filepath.Ext(chapter.Path) != ".cbz" {
		RespondWithError(w, http.StatusUnsupportedMediaType, "Unsupported archive type, only .cbz is supported")
		return
	}

	zipReader, err := zip.OpenReader(chapter.Path)
	if err != nil {
		log.Printf("Error opening zip file %s: %v", chapter.Path, err)
		RespondWithError(w, http.StatusInternalServerError, "Could not open manga archive")
		return
	}
	defer zipReader.Close()

	// Find all image files in the archive and sort them
	var imageFiles []*zip.File
	for _, file := range zipReader.File {
		if !file.FileInfo().IsDir() && library.IsImageFile(file.Name) {
			imageFiles = append(imageFiles, file)
		}
	}
	sort.Slice(imageFiles, func(i, j int) bool {
		return imageFiles[i].Name < imageFiles[j].Name
	})

	// Check if the requested page is out of bounds
	if pageIndex < 0 || pageIndex >= len(imageFiles) {
		RespondWithError(w, http.StatusNotFound, "Page not found in archive")
		return
	}

	// Open the specific image file from the archive
	imageFile := imageFiles[pageIndex]
	rc, err := imageFile.Open()
	if err != nil {
		log.Printf("Error opening image %s from archive %s: %v", imageFile.Name, chapter.Path, err)
		RespondWithError(w, http.StatusInternalServerError, "Could not read page from archive")
		return
	}
	defer rc.Close()

	// Set the correct Content-Type header based on image extension
	ext := filepath.Ext(imageFile.Name)
	contentType := "application/octet-stream" // fallback
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	}
	w.Header().Set("Content-Type", contentType)

	// Stream the file content to the response
	io.Copy(w, rc)
}

// handleGetChapterDetails retrieves and returns details for a single chapter.
func (s *Server) handleGetChapterDetails(w http.ResponseWriter, r *http.Request) {
	chapterIDStr := chi.URLParam(r, "chapterID")
	chapterID, err := strconv.ParseInt(chapterIDStr, 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid chapter ID")
		return
	}

	chapter, err := s.store.GetChapterByID(chapterID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "Chapter not found")
		return
	}

	RespondWithJSON(w, http.StatusOK, chapter)
}

// handleUpdateProgress handles requests to update the progress for a chapter.
func (s *Server) handleUpdateProgress(w http.ResponseWriter, r *http.Request) {
	chapterIDStr := chi.URLParam(r, "chapterID")
	chapterID, err := strconv.ParseInt(chapterIDStr, 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid chapter ID")
		return
	}

	var payload struct {
		ProgressPercent int  `json:"progress_percent"`
		Read            bool `json:"read"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err = s.store.UpdateChapterProgress(chapterID, payload.ProgressPercent, payload.Read)
	if err != nil {
		log.Printf("Failed to update progress for chapter %d: %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to update progress")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	seriesID, _ := strconv.ParseInt(chi.URLParam(r, "seriesID"), 10, 64)
	var payload struct {
		SortBy  string `json:"sort_by"`
		SortDir string `json:"sort_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := s.store.UpdateSeriesSettings(seriesID, payload.SortBy, payload.SortDir); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to update settings")
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleGetChapterNeighbors(w http.ResponseWriter, r *http.Request) {
	seriesID, _ := strconv.ParseInt(chi.URLParam(r, "seriesID"), 10, 64)
	chapterID, _ := strconv.ParseInt(chi.URLParam(r, "chapterID"), 10, 64)

	neighbors, err := s.store.GetChapterNeighbors(seriesID, chapterID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to calculate neighbors")
		return
	}
	RespondWithJSON(w, http.StatusOK, neighbors)
}

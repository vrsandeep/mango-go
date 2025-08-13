package api

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/library"
)

// getListParams extracts all query params for list endpoints.
func getListParams(r *http.Request) (page, perPage int, search, sortBy, sortDir string) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ = strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage <= 0 {
		perPage = 100
	}
	search = r.URL.Query().Get("search")
	sortBy = r.URL.Query().Get("sort_by")
	sortDir = r.URL.Query().Get("sort_dir")
	return
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
	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	chapter, err := s.store.GetChapterByID(chapterID, user.ID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "Chapter not found")
		return
	}

	// --- Image Extraction Logic ---
	// Check if the archive type is supported
	if !library.IsSupportedArchive(chapter.Path) {
		RespondWithError(w, http.StatusUnsupportedMediaType, "Unsupported archive type")
		return
	}

	// Extract the specific page from the archive
	pageData, fileName, err := library.GetPageFromArchive(chapter.Path, pageIndex)
	if err != nil {
		log.Printf("Error extracting page %d from archive %s: %v", pageIndex, chapter.Path, err)
		if strings.Contains(err.Error(), "out of bounds") {
			RespondWithError(w, http.StatusNotFound, "Page not found in archive")
		} else {
			RespondWithError(w, http.StatusInternalServerError, "Could not read page from archive")
		}
		return
	}

	// Set the correct Content-Type header based on image extension
	ext := filepath.Ext(fileName)
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

	// Write the page data to the response
	w.Write(pageData)
}

// handleGetChapterDetails retrieves and returns details for a single chapter.
func (s *Server) handleGetChapterDetails(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	chapterIDStr := chi.URLParam(r, "chapterID")
	chapterID, err := strconv.ParseInt(chapterIDStr, 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid chapter ID")
		return
	}

	chapter, err := s.store.GetChapterByID(chapterID, user.ID)
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

	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var payload struct {
		ProgressPercent float32 `json:"progress_percent"`
		Read            bool    `json:"read"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		log.Printf("Invalid request payload: %v", err)
		return
	}

	err = s.store.UpdateChapterProgress(chapterID, user.ID, int(payload.ProgressPercent), payload.Read)
	if err != nil {
		log.Printf("Failed to update progress for chapter %d: %v", chapterID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to update progress")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleGetChapterNeighbors(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	folderID, _ := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	chapterID, _ := strconv.ParseInt(chi.URLParam(r, "chapterID"), 10, 64)

	neighbors, err := s.store.GetChapterNeighbors(folderID, chapterID, user.ID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to calculate neighbors")
		return
	}
	RespondWithJSON(w, http.StatusOK, neighbors)
}

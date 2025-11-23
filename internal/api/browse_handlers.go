package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
)

func (s *Server) handleBrowseFolder(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	page, perPage, search, sortBy, sortDir := getListParams(r)

	folderIDStr := r.URL.Query().Get("folderId")
	var folderID int64 = 0 // Default to root
	if id, err := strconv.ParseInt(folderIDStr, 10, 64); err == nil {
		folderID = id
	}

	var tagIdStr = r.URL.Query().Get("tagId")
	var tagID *int64
	if tagIdStr != "" {
		id, err := strconv.ParseInt(tagIdStr, 10, 64)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid tag ID")
			return
		}
		tagID = &id
	}

	opts := store.ListItemsOptions{
		UserID:   user.ID,
		ParentID: &folderID,
		Page:     page,
		TagID:    tagID,
		PerPage:  perPage,
		Search:   search,
		SortBy:   sortBy,
		SortDir:  sortDir,
	}
	folder, subfolders, chapters, total, err := s.store.ListItems(opts)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve library contents")
		return
	}

	// Combine results into a single response payload
	response := map[string]interface{}{
		"current_folder": folder,
		"subfolders":     subfolders,
		"chapters":       chapters,
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	RespondWithJSON(w, http.StatusOK, response)
}

func (s *Server) handleGetBreadcrumb(w http.ResponseWriter, r *http.Request) {
	folderIDStr := r.URL.Query().Get("folderId")
	if folderIDStr == "" {
		// No ID means we are at the root, return an empty breadcrumb
		RespondWithJSON(w, http.StatusOK, []*models.Folder{})
		return
	}

	folderID, _ := strconv.ParseInt(folderIDStr, 10, 64)
	breadcrumb, err := s.store.GetFolderPath(folderID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve breadcrumb path")
		return
	}
	RespondWithJSON(w, http.StatusOK, breadcrumb)
}

func (s *Server) handleAddTagToFolder(w http.ResponseWriter, r *http.Request) {
	folderId, _ := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if payload.Name == "" {
		RespondWithError(w, http.StatusBadRequest, "Tag name cannot be empty")
		return
	}

	tag, err := s.store.AddTagToFolder(folderId, payload.Name)
	if err != nil {
		log.Printf("Failed to add tag to folder %d: %v", folderId, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to add tag to folder")
		return
	}
	RespondWithJSON(w, http.StatusCreated, tag)
}

func (s *Server) handleRemoveTagFromFolder(w http.ResponseWriter, r *http.Request) {
	folderID, _ := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	tagID, _ := strconv.ParseInt(chi.URLParam(r, "tagID"), 10, 64)

	if err := s.store.RemoveTagFromFolder(folderID, tagID); err != nil {
		log.Printf("Failed to remove tag %d from folder %d: %v", tagID, folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to remove tag from folder")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateFolderSettings(w http.ResponseWriter, r *http.Request) {
	folderID, _ := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)

	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var payload struct {
		SortBy  string `json:"sort_by"`
		SortDir string `json:"sort_dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := s.store.UpdateFolderSettings(folderID, user.ID, payload.SortBy, payload.SortDir); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to update settings")
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleGetFolderSettings(w http.ResponseWriter, r *http.Request) {
	folderID, _ := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)

	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	settings, err := s.store.GetFolderSettings(folderID, user.ID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve folder settings")
		return
	}

	RespondWithJSON(w, http.StatusOK, settings)
}

// handleMarkFolderAs marks all chapters in a series as read or unread.
func (s *Server) handleMarkFolderAs(w http.ResponseWriter, r *http.Request) {
	folderIDStr := chi.URLParam(r, "folderID")
	folderID, err := strconv.ParseInt(folderIDStr, 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}
	user := getUserFromContext(r)

	var payload struct {
		Read bool `json:"read"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if err := s.store.MarkFolderChaptersAs(folderID, payload.Read, user.ID); err != nil {
		log.Printf("Failed to mark all chapters for folder %d: %v", folderID, err)
		RespondWithError(w, http.StatusInternalServerError, "Failed to update chapters")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// handleUploadFolderCover handles requests to upload a cover image for a folder.
func (s *Server) handleUploadFolderCover(w http.ResponseWriter, r *http.Request) {
	folderID, err := strconv.ParseInt(chi.URLParam(r, "folderID"), 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid folder ID")
		return
	}

	// Limit the upload size to prevent abuse (e.g., 10MB)
	r.Body = http.MaxBytesReader(w, r.Body, 10*1024*1024)

	file, _, err := r.FormFile("cover_file")
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid file upload")
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to read uploaded file")
		return
	}

	thumbnailDataURI, err := library.GenerateThumbnail(fileBytes)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Unsupported image format or corrupt file")
		return
	}

	if err := s.store.UpdateFolderThumbnail(folderID, thumbnailDataURI); err != nil {
		if err == store.ErrFolderNotFound {
			RespondWithError(w, http.StatusNotFound, "Folder not found")
			return
		}
		RespondWithError(w, http.StatusInternalServerError, "Failed to save new cover")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Cover updated successfully."})
}

// handleListAllFolders returns a simple list of all folders for subscription folder selection
func (s *Server) handleListAllFolders(w http.ResponseWriter, r *http.Request) {
	folders, err := s.store.GetAllFoldersByPath()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve folders")
		return
	}

	// Convert to a simple list format with relative paths
	libraryPath := s.app.Config().Library.Path
	var folderList []map[string]interface{}
	for _, folder := range folders {
		// Convert full path to relative path
		relativePath := folder.Path
		if strings.HasPrefix(folder.Path, libraryPath) {
			relativePath = strings.TrimPrefix(folder.Path, libraryPath)
			relativePath = strings.TrimPrefix(relativePath, "/")
		}

		folderList = append(folderList, map[string]interface{}{
			"id":   folder.ID,
			"path": relativePath,
			"name": folder.Name,
		})
	}

	RespondWithJSON(w, http.StatusOK, folderList)
}

// handleSearchFolders searches folders by name and returns matching results
func (s *Server) handleSearchFolders(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		RespondWithJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	folders, err := s.store.SearchFoldersByName(query, 20)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve folders")
		return
	}

	var results []map[string]interface{}
	libraryPath := s.app.Config().Library.Path

	for _, folder := range folders {
			// Convert full path to relative path
			relativePath := folder.Path
			if strings.HasPrefix(folder.Path, libraryPath) {
				relativePath = strings.TrimPrefix(folder.Path, libraryPath)
				relativePath = strings.TrimPrefix(relativePath, "/")
			}

			results = append(results, map[string]interface{}{
				"id":   folder.ID,
				"path": relativePath,
				"name": folder.Name,
			})
	}

	RespondWithJSON(w, http.StatusOK, results)
}

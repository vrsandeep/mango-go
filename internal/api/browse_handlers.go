package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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
	folderId, _ := strconv.ParseInt(chi.URLParam(r, "folderId"), 10, 64)
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
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

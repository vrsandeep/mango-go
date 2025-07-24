package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/store"
)

func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	tags, err := s.store.ListTagsWithCounts()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve tags")
		return
	}
	RespondWithJSON(w, http.StatusOK, tags)
}

func (s *Server) handleGetTagDetails(w http.ResponseWriter, r *http.Request) {
	tagID, _ := strconv.ParseInt(chi.URLParam(r, "tagID"), 10, 64)
	tag, err := s.store.GetTagByID(tagID)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "Tag not found")
		return
	}
	RespondWithJSON(w, http.StatusOK, tag)
}

// func (s *Server) handleListSeriesByTag(w http.ResponseWriter, r *http.Request) {
// 	tagID, _ := strconv.ParseInt(chi.URLParam(r, "tagID"), 10, 64)
// 	page, perPage, search, sortBy, sortDir := getListParams(r)

// 	user := getUserFromContext(r)
// 	if user == nil {
// 		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
// 		return
// 	}

// 	series, total, err := s.store.ListSeriesByTagID(tagID, user.ID, page, perPage, search, sortBy, sortDir)
// 	if err != nil {
// 		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve series for this tag")
// 		return
// 	}

// 	w.Header().Set("X-Total-Count", strconv.Itoa(total))
// 	RespondWithJSON(w, http.StatusOK, series)
// }

// handleListFoldersByTag serves a list of folders (series) that have a specific tag.
func (s *Server) handleListFoldersByTag(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	page, perPage, search, sortBy, sortDir := getListParams(r)
	tagID, _ := strconv.ParseInt(chi.URLParam(r, "tagID"), 10, 64)

	// Use the new generic ListItems function with the TagID option.
	opts := store.ListItemsOptions{
		UserID:  user.ID,
		TagID:   &tagID,
		Page:    page,
		PerPage: perPage,
		Search:  search,
		SortBy:  sortBy,
		SortDir: sortDir,
	}
	// We only care about the folders returned for a tag page.
	_, folders, _, total, err := s.store.ListItems(opts)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve folders for this tag")
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	RespondWithJSON(w, http.StatusOK, folders)
}

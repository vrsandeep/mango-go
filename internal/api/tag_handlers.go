package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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

func (s *Server) handleListSeriesByTag(w http.ResponseWriter, r *http.Request) {
	tagID, _ := strconv.ParseInt(chi.URLParam(r, "tagID"), 10, 64)
	page, perPage, search, sortBy, sortDir := getListParams(r)

	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	series, total, err := s.store.ListSeriesByTagID(tagID, user.ID, page, perPage, search, sortBy, sortDir)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve series for this tag")
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	RespondWithJSON(w, http.StatusOK, series)
}

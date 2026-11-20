// A handler file for all downloader-related API endpoints.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/downloader"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/models"
)

func (s *Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	providerList := providers.GetAll()
	RespondWithJSON(w, http.StatusOK, providerList)
}

func (s *Server) handleProviderSearch(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "providerID")
	query := r.URL.Query().Get("q")

	provider, ok := providers.Get(providerID)
	if !ok {
		RespondWithError(w, http.StatusNotFound, "Provider not found")
		return
	}

	results, err := provider.Search(query)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to perform search")
		return
	}

	RespondWithJSON(w, http.StatusOK, results)
}

func (s *Server) handleProviderGetChapters(w http.ResponseWriter, r *http.Request) {
	providerID := chi.URLParam(r, "providerID")
	// The series identifier might contain special characters (like '/') so it needs to be decoded.
	seriesIdentifier, _ := url.PathUnescape(chi.URLParam(r, "seriesIdentifier"))

	provider, ok := providers.Get(providerID)
	if !ok {
		RespondWithError(w, http.StatusNotFound, "Provider not found")
		return
	}

	results, err := provider.GetChapters(seriesIdentifier)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to get chapters")
		return
	}

	RespondWithJSON(w, http.StatusOK, results)
}

// ChapterQueuePayload is the expected structure for queuing chapters.
type ChapterQueuePayload struct {
	SeriesTitle string                 `json:"series_title"`
	ProviderID  string                 `json:"provider_id"`
	Chapters    []models.ChapterResult `json:"chapters"`
}

func (s *Server) handleAddChaptersToQueue(w http.ResponseWriter, r *http.Request) {
	var payload ChapterQueuePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if len(payload.Chapters) == 0 {
		RespondWithError(w, http.StatusBadRequest, "No chapters provided to queue")
		return
	}

	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	err := s.store.AddChaptersToQueueForUser(user.ID, payload.SeriesTitle, payload.ProviderID, payload.Chapters)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to add chapters to download queue")
		return
	}

	RespondWithJSON(w, http.StatusAccepted, map[string]string{
		"message": fmt.Sprintf("%d chapters have been added to the download queue.", len(payload.Chapters)),
	})
}

func (s *Server) handleGetDownloadQueue(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.GetDownloadQueueFor(actorFromRequest(r))
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve download queue")
		return
	}
	RespondWithJSON(w, http.StatusOK, items)
}

func (s *Server) handleQueueAction(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	actor := actorFromRequest(r)
	switch payload.Action {
	case "pause_all":
		if actor.Admin {
			downloader.PauseDownloads()
		}
		s.store.PauseAllQueueItemsFor(actor)
	case "resume_all":
		if actor.Admin {
			downloader.ResumeDownloads()
		}
		s.store.ResumeAllQueueItemsFor(actor)
	case "retry_failed":
		s.store.ResetFailedQueueItemsFor(actor)
	case "delete_completed":
		s.store.DeleteCompletedQueueItemsFor(actor)
	case "empty_queue":
		s.store.EmptyQueueFor(actor)
	default:
		RespondWithError(w, http.StatusBadRequest, "Invalid action")
		return
	}
	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleQueueItemAction(w http.ResponseWriter, r *http.Request) {
	itemID, err := strconv.ParseInt(chi.URLParam(r, "itemID"), 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	var payload struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	switch payload.Action {
	case "delete", "pause", "resume", "retry":
	default:
		RespondWithError(w, http.StatusBadRequest, "Invalid action")
		return
	}
	if err := s.store.QueueItemVisible(actorFromRequest(r), itemID); err != nil {
		RespondWithError(w, http.StatusNotFound, "Queue item not found")
		return
	}

	switch payload.Action {
	case "delete":
		err = s.store.DeleteQueueItem(itemID)
	case "pause":
		err = downloader.PauseQueueItem(s.app, s.store, itemID)
	case "resume":
		err = downloader.ResumeQueueItem(s.app, s.store, itemID)
	case "retry":
		err = downloader.RetryQueueItem(s.app, s.store, itemID)
	}

	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to perform action")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

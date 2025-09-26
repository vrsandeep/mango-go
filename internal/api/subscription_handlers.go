package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/subscription"
	"github.com/vrsandeep/mango-go/internal/util"
)

func (s *Server) handleSubscribeToSeries(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SeriesTitle      string  `json:"series_title"`
		SeriesIdentifier string  `json:"series_identifier"`
		ProviderID       string  `json:"provider_id"`
		FolderPath       *string `json:"folder_path,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate folder path if provided
	if payload.FolderPath != nil && *payload.FolderPath != "" {
		// Sanitize the folder path
		sanitizedPath := util.SanitizeFolderPath(*payload.FolderPath)
		if sanitizedPath == "" {
			RespondWithError(w, http.StatusBadRequest, "Invalid folder path")
			return
		}

		// Validate the folder path by combining with library path
		basePath := s.app.Config().Library.Path
		if err := util.ValidateFolderPath(sanitizedPath, basePath); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid folder path: %v", err))
			return
		}

		// Store the relative path (not the full path)
		payload.FolderPath = &sanitizedPath
	}

	sub, err := s.store.SubscribeToSeriesWithFolder(payload.SeriesTitle, payload.SeriesIdentifier, payload.ProviderID, payload.FolderPath)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create subscription")
		return
	}

	RespondWithJSON(w, http.StatusCreated, sub)
}

func (s *Server) handleListSubscriptions(w http.ResponseWriter, r *http.Request) {
	providerID := r.URL.Query().Get("provider_id")
	subs, err := s.store.GetAllSubscriptions(providerID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve subscriptions")
		return
	}

	// Convert any full paths to relative paths for consistency
	libraryPath := s.app.Config().Library.Path
	for _, sub := range subs {
		if sub.FolderPath != nil && *sub.FolderPath != "" {
			if strings.HasPrefix(*sub.FolderPath, libraryPath) {
				relativePath := strings.TrimPrefix(*sub.FolderPath, libraryPath)
				relativePath = strings.TrimPrefix(relativePath, "/")
				sub.FolderPath = &relativePath
			}
		}
	}

	RespondWithJSON(w, http.StatusOK, subs)
}

func (s *Server) handleDeleteSubscription(w http.ResponseWriter, r *http.Request) {
	subID, _ := strconv.ParseInt(chi.URLParam(r, "subID"), 10, 64)
	if err := s.store.DeleteSubscription(subID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUpdateSubscriptionFolderPath(w http.ResponseWriter, r *http.Request) {
	subID, _ := strconv.ParseInt(chi.URLParam(r, "subID"), 10, 64)

	var payload struct {
		FolderPath *string `json:"folder_path,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Validate folder path if provided
	if payload.FolderPath != nil && *payload.FolderPath != "" {
		// Sanitize the folder path
		sanitizedPath := util.SanitizeFolderPath(*payload.FolderPath)
		if sanitizedPath == "" {
			RespondWithError(w, http.StatusBadRequest, "Invalid folder path")
			return
		}

		// Validate the folder path by combining with library path
		basePath := s.app.Config().Library.Path
		if err := util.ValidateFolderPath(sanitizedPath, basePath); err != nil {
			RespondWithError(w, http.StatusBadRequest, fmt.Sprintf("Invalid folder path: %v", err))
			return
		}

		// Update the payload with the sanitized path
		payload.FolderPath = &sanitizedPath
	}

	err := s.store.UpdateSubscriptionFolderPath(subID, payload.FolderPath)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			RespondWithError(w, http.StatusNotFound, "Subscription not found")
		} else {
			RespondWithError(w, http.StatusInternalServerError, "Failed to update subscription folder path")
		}
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Subscription folder path updated successfully."})
}

func (s *Server) handleRecheckSubscription(w http.ResponseWriter, r *http.Request) {
	subID, _ := strconv.ParseInt(chi.URLParam(r, "subID"), 10, 64)

	// Run the check in a background goroutine so the API call returns immediately.
	go func() {
		subService := subscription.NewService(s.app)
		subService.CheckSingleSubscription(subID)
	}()

	RespondWithJSON(w, http.StatusAccepted, map[string]string{"message": "Re-check has been initiated."})
}

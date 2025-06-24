package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/subscription"
)

func (s *Server) handleSubscribeToSeries(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		SeriesTitle      string `json:"series_title"`
		SeriesIdentifier string `json:"series_identifier"`
		ProviderID       string `json:"provider_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	sub, err := s.store.SubscribeToSeries(payload.SeriesTitle, payload.SeriesIdentifier, payload.ProviderID)
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

func (s *Server) handleRecheckSubscription(w http.ResponseWriter, r *http.Request) {
	subID, _ := strconv.ParseInt(chi.URLParam(r, "subID"), 10, 64)

	// Run the check in a background goroutine so the API call returns immediately.
	go func() {
		subService := subscription.NewService(s.app)
		subService.CheckSingleSubscription(subID)
	}()

	RespondWithJSON(w, http.StatusAccepted, map[string]string{"message": "Re-check has been initiated."})
}

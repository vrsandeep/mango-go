package api

import (
	"encoding/json"
	"net/http"
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

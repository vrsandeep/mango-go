// A NEW file with helper functions for sending standardized JSON responses.

package api

import (
	"encoding/json"
	"net/http"
)

// RespondWithJSON writes a JSON response with the given status code and payload.
func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		// If marshaling fails, return an error response
		RespondWithError(w, http.StatusInternalServerError, "Failed to marshal response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// RespondWithError writes a standardized JSON error response.
func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, map[string]string{"error": message})
}

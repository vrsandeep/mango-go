// A handler file for all downloader-related API endpoints.

package api

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
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

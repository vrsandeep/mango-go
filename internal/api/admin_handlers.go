package api

import (
	"net/http"

	"github.com/vrsandeep/mango-go/internal/jobs"
)

func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	RespondWithJSON(w, http.StatusOK, map[string]string{"version": s.app.Version})
}

func (s *Server) handleScanLibrary(w http.ResponseWriter, r *http.Request) {
	go jobs.RunFullScan(s.app)
	RespondWithJSON(w, http.StatusAccepted, map[string]string{"message": "Full library scan started."})
}

func (s *Server) handleScanMissing(w http.ResponseWriter, r *http.Request) {
	go jobs.RunIncrementalScan(s.app)
	RespondWithJSON(w, http.StatusAccepted, map[string]string{"message": "Incremental scan for missing items started."})
}

func (s *Server) handlePruneDatabase(w http.ResponseWriter, r *http.Request) {
	go jobs.RunPruneDatabase(s.app)
	RespondWithJSON(w, http.StatusAccepted, map[string]string{"message": "Database pruning started."})
}

func (s *Server) handleGenerateThumbnails(w http.ResponseWriter, r *http.Request) {
	go jobs.RunThumbnailGeneration(s.app)
	RespondWithJSON(w, http.StatusAccepted, map[string]string{"message": "Thumbnail regeneration started."})
}

package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	RespondWithJSON(w, http.StatusOK, map[string]string{"version": s.app.Version})
}


func (s *Server) handleRunAdminJob(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		JobName string `json:"job_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err := s.app.JobManager().RunJob(payload.JobName, s.app)
	if err != nil {
		RespondWithError(w, http.StatusConflict, err.Error()) // 409 Conflict if a job is already running
		return
	}

	RespondWithJSON(w, http.StatusAccepted, map[string]string{
		"message": "Job '" + payload.JobName + "' started successfully.",
	})
}

func (s *Server) handleGetAdminJobsStatus(w http.ResponseWriter, r *http.Request) {
	statuses := s.app.JobManager().GetStatus()
	RespondWithJSON(w, http.StatusOK, statuses)
}
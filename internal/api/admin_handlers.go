package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/vrsandeep/mango-go/internal/store"
)

func (s *Server) handleGetVersion(w http.ResponseWriter, r *http.Request) {
	RespondWithJSON(w, http.StatusOK, map[string]string{"version": s.app.Version})
}

func (s *Server) handleRunAdminJob(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	err := s.app.JobManager().RunJob(payload.JobID, s.app)
	if err != nil {
		RespondWithError(w, http.StatusConflict, err.Error()) // 409 Conflict if a job is already running
		return
	}

	RespondWithJSON(w, http.StatusAccepted, map[string]string{
		"message": "Job '" + payload.JobID + "' started successfully.",
	})
}

func (s *Server) handleGetAdminJobsStatus(w http.ResponseWriter, r *http.Request) {
	statuses := s.app.JobManager().GetStatus()
	RespondWithJSON(w, http.StatusOK, statuses)
}

// handleGetBadFiles retrieves all bad files from the database
func (s *Server) handleGetBadFiles(w http.ResponseWriter, r *http.Request) {
	badFileStore := store.NewBadFileStore(s.app.DB())

	badFiles, err := badFileStore.GetAllBadFiles()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve bad files: %v", err))
		return
	}
	RespondWithJSON(w, http.StatusOK, badFiles)
}

// handleDeleteBadFile removes a bad file entry by ID
func (s *Server) handleDeleteBadFile(w http.ResponseWriter, r *http.Request) {
	// Extract ID from URL path
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		RespondWithError(w, http.StatusBadRequest, "Missing file ID")
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid file ID")
		return
	}

	badFileStore := store.NewBadFileStore(s.app.DB())
	err = badFileStore.DeleteBadFile(id)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete bad file: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Bad file entry deleted successfully"})
}

// handleDownloadBadFilesCSV downloads the list of bad files as a CSV file
func (s *Server) handleDownloadBadFilesCSV(w http.ResponseWriter, r *http.Request) {
	badFileStore := store.NewBadFileStore(s.app.DB())

	badFiles, err := badFileStore.GetAllBadFiles()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve bad files: %v", err))
		return
	}

	// Set response headers for CSV download
	filename := fmt.Sprintf("bad_files_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// Create CSV writer
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write CSV header
	header := []string{"ID", "File Name", "Path", "Error", "File Size (bytes)", "Detected At", "Last Checked"}
	if err := csvWriter.Write(header); err != nil {
		http.Error(w, "Failed to write CSV header", http.StatusInternalServerError)
		return
	}

	// Write data rows
	for _, bf := range badFiles {
		row := []string{
			strconv.FormatInt(bf.ID, 10),
			bf.FileName,
			bf.Path,
			bf.Error,
			strconv.FormatInt(bf.FileSize, 10),
			bf.DetectedAt.Format("2006-01-02 15:04:05"),
			bf.LastChecked.Format("2006-01-02 15:04:05"),
		}
		if err := csvWriter.Write(row); err != nil {
			http.Error(w, "Failed to write CSV row", http.StatusInternalServerError)
			return
		}
	}
}

// handleGetBadFilesCount returns the count of bad files
func (s *Server) handleGetBadFilesCount(w http.ResponseWriter, r *http.Request) {
	badFileStore := store.NewBadFileStore(s.app.DB())

	count, err := badFileStore.CountBadFiles()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to count bad files: %v", err))
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"count": count,
		"show_download": count > 50,
	})
}

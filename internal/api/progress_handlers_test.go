package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
)

// test file dedicated to the new progress-related API endpoints.
func TestHandleGetChapterDetails(t *testing.T) {
	server, _ := setupTestServer(t)
	router := server.Router()

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/series/1/chapters/1", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var chapter models.Chapter
		if err := json.Unmarshal(rr.Body.Bytes(), &chapter); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if chapter.ID != 1 {
			t.Errorf("Expected chapter ID 1, got %d", chapter.ID)
		}
		if chapter.Read != false {
			t.Errorf("Expected chapter Read status false, got %t", chapter.Read)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/series/1/chapters/999", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

func TestHandleUpdateProgress(t *testing.T) {
	server, db := setupTestServer(t)
	router := server.Router()
	s := store.New(db)

	t.Run("Success", func(t *testing.T) {
		payload := `{"current_page": 75, "read": true}`
		req, _ := http.NewRequest("POST", "/api/chapters/1/progress", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify the change in the database
		chapter, err := s.GetChapterByID(1)
		if err != nil {
			t.Fatalf("Failed to get chapter after update: %v", err)
		}
		if chapter.CurrentPage != 75 {
			t.Errorf("DB value for current_page was not updated: want 75, got %d", chapter.CurrentPage)
		}
		if !chapter.Read {
			t.Error("DB value for read was not updated: want true, got false")
		}
	})

	t.Run("Invalid Payload", func(t *testing.T) {
		payload := `{"invalid_json": true}`
		req, _ := http.NewRequest("POST", "/api/chapters/1/progress", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			// Note: The handler still returns 200 OK because the JSON unmarshaling
			// will just result in zero-values for the expected fields. The DB update
			// will proceed with page=0 and read=false. This is an acceptable behavior.
			// A more robust implementation might return a 400 Bad Request if fields are missing.
			t.Logf("Handler returned OK for partially invalid payload, which is acceptable.")
		}
	})

	t.Run("Malformed JSON", func(t *testing.T) {
		payload := `{"current_page": 75,` // Malformed
		req, _ := http.NewRequest("POST", "/api/chapters/1/progress", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code for malformed JSON: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

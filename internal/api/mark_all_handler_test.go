package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
)

func TestHandleMarkAllAs(t *testing.T) {
	server, db := setupTestServer(t)
	router := server.Router()
	s := store.New(db)

	t.Run("Mark all as read", func(t *testing.T) {
		payload := `{"read": true}`
		req, _ := http.NewRequest("POST", "/api/series/1/mark-all-as", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify the change in the database
		series, err := s.GetSeriesByID(1, 1, 10) // Get all chapters
		if err != nil {
			t.Fatalf("Failed to get series after update: %v", err)
		}
		for _, chapter := range series.Chapters {
			if !chapter.Read {
				t.Errorf("Expected chapter %d to be marked as read, but it was not", chapter.ID)
			}
			if chapter.ProgressPercent != 100 {
				t.Errorf("Expected chapter %d progress to be 100, but it was %d", chapter.ID, chapter.ProgressPercent)
			}
		}
	})

	t.Run("Mark all as unread", func(t *testing.T) {
		// First, mark them as read to ensure the "unread" call works
		s.MarkAllChaptersAs(1, true)

		payload := `{"read": false}`
		req, _ := http.NewRequest("POST", "/api/series/1/mark-all-as", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify the change in the database
		series, err := s.GetSeriesByID(1, 1, 10) // Get all chapters
		if err != nil {
			t.Fatalf("Failed to get series after update: %v", err)
		}
		for _, chapter := range series.Chapters {
			if chapter.Read {
				t.Errorf("Expected chapter %d to be marked as unread, but it was not", chapter.ID)
			}
			if chapter.ProgressPercent != 0 {
				t.Errorf("Expected chapter %d progress to be 0, but it was %d", chapter.ID, chapter.ProgressPercent)
			}
		}
	})
}

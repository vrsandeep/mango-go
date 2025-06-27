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
		req.AddCookie(CookieForUser(t, server, "testuser", "password", "user"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v, %s", status, http.StatusOK, rr.Body.String())
		}

		// Verify the change in the database
		series, count, err := s.GetSeriesByID(1, 1, 1, 10, "", "", "") // Get all chapters
		if err != nil {
			t.Fatalf("Failed to get series: %v", err)
		}
		if count != 1 {
			t.Fatalf("Expected 1 chapter, got %d", count)
		}
		if len(series.Chapters) != 1 {
			t.Fatalf("Expected 1 chapter, got %d", len(series.Chapters))
		}

		// Verify all chapters are unread initially
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
		s.MarkAllChaptersAs(1, true, 1)

		payload := `{"read": false}`
		req, _ := http.NewRequest("POST", "/api/series/1/mark-all-as", bytes.NewBufferString(payload))
		req.AddCookie(CookieForUser(t, server, "testuser", "password", "user"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify the change in the database
		series, count, err := s.GetSeriesByID(1, 1, 1, 10, "", "", "") // Get all chapters
		if err != nil {
			t.Fatalf("Failed to get series: %v", err)
		}
		if count != 1 {
			t.Fatalf("Expected 1 chapter, got %d", count)
		}
		if len(series.Chapters) != 1 {
			t.Fatalf("Expected 1 chapters, got %d", len(series.Chapters))
		}

		// Verify all chapters are unread initially
		for _, chapter := range series.Chapters {
			if chapter.Read {
				t.Errorf("Expected chapter %d to be unread", chapter.ID)
			}
			if chapter.ProgressPercent != 0 {
				t.Errorf("Expected chapter %d to have 0 progress", chapter.ID)
			}
		}
	})
}

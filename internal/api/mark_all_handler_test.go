package api_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestHandleMarkAllAs(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	s := store.New(db)
	testutil.PersistOneFolderAndChapter(t, db)

	t.Run("Mark all as read", func(t *testing.T) {
		payload := `{"read": true}`
		req, _ := http.NewRequest("POST", "/api/folders/1/mark-all-as", bytes.NewBufferString(payload))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v, %s", status, http.StatusOK, rr.Body.String())
		}

		// Verify the change in the database
		_, _, chapters, _, err := s.ListItems(store.ListItemsOptions{
			UserID:   1,
			ParentID: &[]int64{1}[0],
			Page:     1,
			PerPage:  10,
			SortBy:   "",
			SortDir:  "",
		}) // Get all chapters
		if err != nil {
			t.Fatalf("Failed to get folder: %v", err)
		}
		if len(chapters) != 1 {
			t.Fatalf("Expected 1 chapter, got %d", len(chapters))
		}

		// Verify all chapters are unread initially
		for _, chapter := range chapters {
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
		s.MarkFolderChaptersAs(1, true, 1)

		payload := `{"read": false}`
		req, _ := http.NewRequest("POST", "/api/folders/1/mark-all-as", bytes.NewBufferString(payload))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify the change in the database
		_, _, chapters, _, err := s.ListItems(store.ListItemsOptions{
			UserID:   1,
			ParentID: &[]int64{1}[0],
			Page:     1,
			PerPage:  10,
			SortBy:   "",
			SortDir:  "",
		}) // Get all chapters
		if err != nil {
			t.Fatalf("Failed to get folder: %v", err)
		}
		if len(chapters) != 1 {
			t.Fatalf("Expected 1 chapters, got %d", len(chapters))
		}

		// Verify all chapters are unread initially
		for _, chapter := range chapters {
			if chapter.Read {
				t.Errorf("Expected chapter %d to be unread", chapter.ID)
			}
			if chapter.ProgressPercent != 0 {
				t.Errorf("Expected chapter %d to have 0 progress", chapter.ID)
			}
		}
	})
}

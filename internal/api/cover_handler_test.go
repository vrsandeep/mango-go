package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
)

func CookieForUser(t *testing.T, server *Server, username, password, role string) *http.Cookie {
	t.Helper()
	cookie := GetAuthCookie(t, server, username, password, role)
	if cookie == nil {
		t.Fatal("Failed to get session cookie after successful login for test user")
	}
	// on cleanup, delete the user
	t.Cleanup(func() {
		user, _ := server.store.GetUserByUsername(username)
		server.store.DeleteUser(user.ID)
	})
	return cookie
}

func TestHandleUpdateCover(t *testing.T) {
	server, db := setupTestServer(t) // This helper is in handlers_test.go
	router := server.Router()
	s := store.New(db)

	t.Run("Success", func(t *testing.T) {
		newCoverURL := "http://example.com/new_cover.jpg"
		payload := `{"url": "` + newCoverURL + `"}`

		req, _ := http.NewRequest("POST", "/api/series/1/cover", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(CookieForUser(t, server, "testuser", "password", "user"))

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Verify the change in the database
		series, count, err := s.GetSeriesByID(1, 1, 1, 1, "", "", "") // page and perPage don't matter here
		if err != nil {
			t.Fatalf("Failed to get series after update: %v", err)
		}
		if count != 1 {
			t.Errorf("Expected 1 chapter after update, got %d", count)
		}
		if series.CustomCoverURL != newCoverURL {
			t.Errorf("DB value for custom_cover_url was not updated: want %s, got %s", newCoverURL, series.CustomCoverURL)
		}
	})

	t.Run("Invalid Series ID", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/series/99x9/cover", bytes.NewBufferString(`{"url": "test"}`))
		req.AddCookie(CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// The store function returns an error which leads to a 500.
		// A more robust implementation might check for existence first and return a 404.
		// For now, we test the current behavior.
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code for non-existent series: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Non existent Series", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/series/9999/cover", bytes.NewBufferString(`{"url": "test"}`))
		req.AddCookie(CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code for non-existent series: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("Missing URL", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/series/1/cover", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code for missing URL: got %v want %v", status, http.StatusBadRequest)
		}
	})

	t.Run("Malformed JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/series/1/cover", bytes.NewBufferString(`{"url":`))
		req.AddCookie(CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("handler returned wrong status code for malformed JSON: got %v want %v", status, http.StatusBadRequest)
		}
	})
}

package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestTagHandlers(t *testing.T) {
	server, db := SetupTestServerWithProviders(t)
	router := server.Router()

	// Add a dummy series and tag for testing
	res, _ := db.Exec(`INSERT INTO series (id, title, path, created_at, updated_at) VALUES (1, 'Test', '/test', ?, ?)`, time.Now(), time.Now())
	seriesID, _ := res.LastInsertId()
	res, _ = db.Exec("INSERT INTO tags (id, name) VALUES (1, 'action')")
	tagID, _ := res.LastInsertId()
	db.Exec("INSERT INTO series_tags (series_id, tag_id) VALUES (?, ?)", seriesID, tagID)

	t.Run("List Tags", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/tags", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("ListTags handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var tags []*models.Tag
		json.Unmarshal(rr.Body.Bytes(), &tags)
		if len(tags) != 1 {
			t.Fatalf("Expected 1 tag, got %d", len(tags))
		}
		if tags[0].Name != "action" {
			t.Errorf("Expected tag name 'action', got %s", tags[0].Name)
		}
		if tags[0].SeriesCount != 1 {
			t.Errorf("Expected series count 1, got %d", tags[0].SeriesCount)
		}
	})

	t.Run("List Series By Tag", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/tags/1/series", nil)
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("ListSeriesByTag handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var series []*models.Series
		json.Unmarshal(rr.Body.Bytes(), &series)
		// Note: This relies on the placeholder logic in the handler. A full test
		// would require the dynamic SQL to be implemented.
		if len(series) < 1 {
			t.Fatalf("Expected at least 1 series for the tag, got %d", len(series))
		}
	})
}

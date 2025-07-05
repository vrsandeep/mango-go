package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestHandleSubscribeToSeries(t *testing.T) {
	server, db := SetupTestServerWithProviders(t)
	router := server.Router()

	t.Run("Success", func(t *testing.T) {
		payload := map[string]string{
			"series_title":      "Subscribe Test",
			"series_identifier": "sub-test-1",
			"provider_id":       "mockadex",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/subscriptions", bytes.NewBuffer(body))
		req.AddCookie(testutil.CookieForUser(t, server, "testuser", "password", "user"))
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
		}

		var count int
		db.QueryRow("SELECT COUNT(*) FROM subscriptions WHERE series_identifier = 'sub-test-1'").Scan(&count)
		if count != 1 {
			t.Error("Expected subscription to be created, but it was not found in DB")
		}
	})
}

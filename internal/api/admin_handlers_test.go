package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminHandlers(t *testing.T) {
	server, _ := setupTestServer(t) // This helper sets up a test server and DB
	router := server.Router()

	testCases := []struct {
		name     string
		endpoint string
		method   string
	}{
		{"Scan Library", "/api/admin/scan-library", "POST"},
		{"Scan Incremental", "/api/admin/scan-incremental", "POST"},
		{"Prune Database", "/api/admin/prune-database", "POST"},
		{"Regenerate Thumbnails", "/api/admin/generate-thumbnails", "POST"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, tc.endpoint, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusAccepted {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
			}
		})
	}

	t.Run("Get Version", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/version", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})
}

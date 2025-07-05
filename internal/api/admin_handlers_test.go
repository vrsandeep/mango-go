package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestAdminHandlers(t *testing.T) {
	server, _ := testutil.SetupTestServer(t) // This helper sets up a test server and DB
	router := server.Router()

	adminCookie := testutil.GetAuthCookie(t, server, "testadmin", "password", "admin")
	userCookie := testutil.GetAuthCookie(t, server, "testuser", "password", "user")

	testCases := []struct {
		name     string
		endpoint string
		method   string
		cookie   *http.Cookie
	}{
		{"Scan Library", "/api/admin/scan-library", "POST", adminCookie},
		{"Scan Incremental", "/api/admin/scan-incremental", "POST", adminCookie},
		{"Prune Database", "/api/admin/prune-database", "POST", adminCookie},
		{"Regenerate Thumbnails", "/api/admin/generate-thumbnails", "POST", adminCookie},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest(tc.method, tc.endpoint, nil)
			req.AddCookie(tc.cookie)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusAccepted {
				t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
			}
		})
	}

	t.Run("Unauthorized Access", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/admin/scan-library", nil)
		req.AddCookie(userCookie) // Use a regular user cookie
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusForbidden {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
		}
	})

	t.Run("Get Version", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/version", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v %s", status, http.StatusOK, rr.Body.String())
		}
	})
}

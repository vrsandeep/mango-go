package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestAdminHandlers(t *testing.T) {
	server, _, jobManager := testutil.SetupTestServer(t) // This helper sets up a test server and DB
	router := server.Router()
	jobManager.Register("test-job", "Test Job", func(ctx jobs.JobContext) {
		time.Sleep(1 * time.Second)
	})

	adminCookie := testutil.GetAuthCookie(t, server, "testadmin", "password", "admin")
	userCookie := testutil.GetAuthCookie(t, server, "testuser", "password", "user")

	t.Run("Library Sync", func(t *testing.T) {
		type body struct {
			JobID string `json:"job_id"`
		}
		payload, _ := json.Marshal(body{JobID: "test-job"})
		req, _ := http.NewRequest("POST", "/api/admin/jobs/run", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusAccepted {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusAccepted)
		}

		req, _ = http.NewRequest("GET", "/api/admin/jobs/status", nil)
		req.AddCookie(adminCookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		var statuses []jobs.JobStatus
		if err := json.NewDecoder(rr.Body).Decode(&statuses); err != nil {
			t.Fatalf("failed to decode response body: %v", err)
		}
		if len(statuses) == 0 {
			t.Fatalf("no jobs found in statuses")
		}

		// Submitting again should return a 409 Conflict
		rr = httptest.NewRecorder()
		req, _ = http.NewRequest("POST", "/api/admin/jobs/run", bytes.NewBuffer(payload))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(adminCookie)
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusConflict {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusConflict)
		}
	})


	t.Run("Unauthorized Access", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/admin/jobs/run", nil)
		req.AddCookie(userCookie) // Use a regular user cookie
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusForbidden {
			t.Fatalf("handler returned wrong status code: got %v want %v", status, http.StatusForbidden)
		}
	})

	t.Run("Get Version", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/version", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v %s", status, http.StatusOK, rr.Body.String())
		}
	})
}

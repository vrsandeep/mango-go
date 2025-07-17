package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestAdminUserHandlers(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Create admin and regular user for testing roles
	adminCookie := testutil.GetAuthCookie(t, server, "testadmin", "password", "admin")
	userCookie := testutil.GetAuthCookie(t, server, "testuser", "password", "user")

	t.Run("Admin can list users", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/admin/users", nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status 200, got %d", status)
		}
	})

	t.Run("Non-admin cannot list users", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/admin/users", nil)
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", status)
		}
	})

	var createdUserID int64
	t.Run("Admin can create a user", func(t *testing.T) {
		payload := `{"username":"newuser","password":"newpassword","role":"user"}`
		req, _ := http.NewRequest("POST", "/api/admin/users", bytes.NewBufferString(payload))
		req.AddCookie(adminCookie)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", status)
		}

		var user models.User
		json.Unmarshal(rr.Body.Bytes(), &user)
		if user.Username != "newuser" {
			t.Error("Created user has wrong username")
		}
		createdUserID = user.ID
	})

	t.Run("Admin can update a user", func(t *testing.T) {
		payload := `{"username":"updateduser","role":"admin"}`
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/admin/users/%d", createdUserID), bytes.NewBufferString(payload))
		req.AddCookie(adminCookie)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status 200, got %d", status)
		}
		query := "SELECT username, role FROM users WHERE id = ?"
		var username, role string
		err := db.QueryRow(query, createdUserID).Scan(&username, &role)
		if err != nil {
			t.Fatalf("Failed to query updated user: %v", err)
		}
		if username != "updateduser" || role != "admin" {
			t.Errorf("Expected updated user to have username 'updateduser' and role 'admin', got username '%s' and role '%s'", username, role)
		}
	})

	t.Run("Admin can delete a user", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/admin/users/%d", createdUserID), nil)
		req.AddCookie(adminCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", status)
		}
	})
}

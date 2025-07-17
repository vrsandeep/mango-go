package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestAuthHandlers(t *testing.T) {
	server, _, _ := testutil.SetupTestServer(t)
	router := server.Router()

	// Pre-create a user for login tests
	testutil.GetAuthCookie(t, server, "testuser", "password123", "user")

	t.Run("Successful Login", func(t *testing.T) {
		payload := `{"username":"testuser", "password":"password123"}`
		req, _ := http.NewRequest("POST", "/api/users/login", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		foundCookie := false
		for _, cookie := range rr.Result().Cookies() {
			if cookie.Name == "session_token" {
				foundCookie = true
				if cookie.Value == "" {
					t.Error("session token cookie is empty")
				}
				if !cookie.HttpOnly {
					t.Error("session cookie is not HttpOnly")
				}
			}
		}
		if !foundCookie {
			t.Error("session_token cookie not found in response")
		}
	})

	t.Run("Login with Wrong Password", func(t *testing.T) {
		payload := `{"username":"testuser", "password":"wrongpassword"}`
		req, _ := http.NewRequest("POST", "/api/users/login", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("Get Me (Authenticated)", func(t *testing.T) {
		// Use the helper to get a valid session cookie
		userCookie := testutil.GetAuthCookie(t, server, "getme_user", "password", "user")

		if userCookie == nil {
			t.Fatal("Failed to get session cookie after successful login for getme_user user")
		}

		req, _ := http.NewRequest("GET", "/api/users/me", nil)
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("handler returned wrong status code: got %v want %v %s", status, http.StatusOK, rr.Body.String())
		}

		var user models.User
		if err := json.Unmarshal(rr.Body.Bytes(), &user); err != nil {
			t.Fatalf("Could not unmarshal response body: %v", err)
		}
		if user.Username != "getme_user" {
			t.Errorf("Expected username 'getme_user', got '%s'", user.Username)
		}
		if user.Role != "user" {
			t.Errorf("Expected role 'user', got '%s'", user.Role)
		}
	})

	t.Run("Get Me (Unauthenticated)", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/users/me", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusUnauthorized)
		}
	})

	t.Run("Successful Logout", func(t *testing.T) {
		userCookie := testutil.GetAuthCookie(t, server, "logout_user", "password", "user")

		req, _ := http.NewRequest("POST", "/api/users/logout", nil)
		req.AddCookie(userCookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		// Check that the cookie is expired
		foundExpiredCookie := false
		for _, cookie := range rr.Result().Cookies() {
			if cookie.Name == "session_token" {
				if cookie.MaxAge < 0 {
					foundExpiredCookie = true
				}
			}
		}
		if !foundExpiredCookie {
			t.Error("session_token cookie was not expired on logout")
		}
	})
}

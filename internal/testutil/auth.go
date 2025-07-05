package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/auth"
)

// GetAuthCookie creates a user, logs them in, and returns a valid session cookie.
func GetAuthCookie(t *testing.T, s *api.Server, username, password, role string) *http.Cookie {
	t.Helper()

	// Step 1: CORRECTLY hash the password before creating the user.
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password for test user: %v", err)
	}
	// The store's CreateUser expects a hash, not a plaintext password.
	_, err = s.Store().CreateUser(username, passwordHash, role)
	if err != nil {
		t.Fatalf("Failed to create test user '%s': %v", username, err)
	}

	// Step 2: Log in as the newly created user to get a session.
	loginPayload := map[string]string{"username": username, "password": password}
	payloadBytes, _ := json.Marshal(loginPayload)
	req, _ := http.NewRequest("POST", "/api/users/login", bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	s.Router().ServeHTTP(rr, req)

	// Assert that the login was successful.
	if status := rr.Code; status != http.StatusOK {
		t.Fatalf("Login failed within test helper for user '%s': got status %d, want 200", username, status)
	}

	// Step 3: Extract the session cookie from the response.
	cookies := rr.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			return cookie
		}
	}

	t.Fatal("Failed to get session cookie after successful login for test user")
	return nil
}

func CookieForUser(t *testing.T, server *api.Server, username, password, role string) *http.Cookie {
	t.Helper()
	cookie := GetAuthCookie(t, server, username, password, role)
	if cookie == nil {
		t.Fatal("Failed to get session cookie after successful login for test user")
	}
	// on cleanup, delete the user
	t.Cleanup(func() {
		user, _ := server.Store().GetUserByUsername(username)
		server.Store().DeleteUser(user.ID)
	})
	return cookie
}

package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/vrsandeep/mango-go/internal/auth"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	ip := clientIP(r)
	if s.loginLimiter != nil && !s.loginLimiter.allow(ip) {
		RespondWithError(w, http.StatusTooManyRequests, "Too many login attempts. Try again later.")
		return
	}

	user, err := s.store.GetUserByUsername(payload.Username)
	if err != nil {
		if s.loginLimiter != nil {
			s.loginLimiter.record(ip)
		}
		RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	if !auth.CheckPasswordHash(payload.Password, user.PasswordHash) {
		if s.loginLimiter != nil {
			s.loginLimiter.record(ip)
		}
		RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	token, err := s.store.CreateSession(user.ID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	http.SetCookie(w, sessionCookie(r, token, time.Now().Add(7*24*time.Hour), 0))

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		s.store.DeleteSession(cookie.Value)
	}

	// Expire the cookie on the client side
	http.SetCookie(w, sessionCookie(r, "", time.Unix(0, 0), -1))
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}
	RespondWithJSON(w, http.StatusOK, user)
}

// sessionCookie builds the session cookie. Secure follows requestIsHTTPS.
// maxAge is applied only when non-zero; logout passes -1 to expire the cookie.
func sessionCookie(r *http.Request, value string, expires time.Time, maxAge int) *http.Cookie {
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    value,
		Expires:  expires,
		Path:     "/",
		HttpOnly: true,
		Secure:   requestIsHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	}
	if maxAge != 0 {
		cookie.MaxAge = maxAge
	}
	return cookie
}

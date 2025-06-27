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

	user, err := s.store.GetUserByUsername(payload.Username)
	if err != nil {
		RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	if !auth.CheckPasswordHash(payload.Password, user.PasswordHash) {
		RespondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	token, err := s.store.CreateSession(user.ID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to create session")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil, // Set secure flag if using HTTPS
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		s.store.DeleteSession(cookie.Value)
	}

	// Expire the cookie on the client side
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil, // Set secure flag if using HTTPS
		SameSite: http.SameSiteLaxMode,
	})
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

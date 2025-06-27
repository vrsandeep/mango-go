package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/vrsandeep/mango-go/internal/auth"
)

func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}
	RespondWithJSON(w, http.StatusOK, users)
}

func (s *Server) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if payload.Username == "" || payload.Password == "" || (payload.Role != "admin" && payload.Role != "user") {
		RespondWithError(w, http.StatusBadRequest, "Username, password, and a valid role are required")
		return
	}

	passwordHash, err := auth.HashPassword(payload.Password)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user, err := s.store.CreateUser(payload.Username, passwordHash, payload.Role)
	if err != nil {
		// Could be a unique constraint violation
		RespondWithError(w, http.StatusConflict, "Username already exists")
		return
	}
	RespondWithJSON(w, http.StatusCreated, user)
}

func (s *Server) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)
	var payload struct {
		Username string `json:"username"`
		Role     string `json:"role"`
		Password string `json:"password,omitempty"` // Password is optional
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Update basic info
	if err := s.store.UpdateUser(userID, payload.Username, payload.Role); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	// Update password if provided
	if payload.Password != "" {
		passwordHash, err := auth.HashPassword(payload.Password)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}
		if err := s.store.UpdateUserPassword(userID, passwordHash); err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Failed to update password")
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.ParseInt(chi.URLParam(r, "userID"), 10, 64)

	// You might want to prevent an admin from deleting themselves
	currentUser := getUserFromContext(r)
	if currentUser.ID == userID {
		RespondWithError(w, http.StatusBadRequest, "Cannot delete your own account")
		return
	}

	if err := s.store.DeleteUser(userID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

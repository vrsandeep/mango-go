package api

// This file contains the middleware for handling authentication and role-based authorization.

import (
	"context"
	"net/http"

	"github.com/vrsandeep/mango-go/internal/models"
)

// contextKey is a private type to prevent collisions with other context keys.
type contextKey string

const userContextKey = contextKey("user")

// AuthMiddleware is a middleware that verifies a user's session.
// If the session is valid, it retrieves the user's details from the database
// and injects them into the request's context for downstream handlers to use.
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			// If no cookie is present, the user is unauthorized.
			RespondWithError(w, http.StatusUnauthorized, "Unauthorized: No session token")
			return
		}

		user, err := s.store.GetUserFromSession(cookie.Value)
		if err != nil {
			// If the token is invalid or expired, the user is unauthorized.
			RespondWithError(w, http.StatusUnauthorized, "Unauthorized: Invalid session")
			return
		}

		// Add the user object to the request context.
		ctx := context.WithValue(r.Context(), userContextKey, user)
		// Call the next handler in the chain with the new context.
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AdminOnlyMiddleware is a middleware that ensures only users with the 'admin' role can access a route.
// It must be chained *after* the AuthMiddleware.
func (s *Server) AdminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := getUserFromContext(r)

		// This should theoretically not happen if AuthMiddleware is used first, but it's a safe check.
		if user == nil {
			RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		if user.Role != "admin" {
			RespondWithError(w, http.StatusForbidden, "Forbidden: Administrator access required")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getUserFromContext is a helper function to safely retrieve the user object from the request context.
// It returns nil if the user is not found in the context.
func getUserFromContext(r *http.Request) *models.User {
	user, ok := r.Context().Value(userContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

package api

import (
	"net/http"

	"github.com/vrsandeep/mango-go/internal/models"
)

func (s *Server) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	items, err := s.store.ListNotifications(user.ID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve notifications")
		return
	}

	hasUnread, err := s.store.HasUnreadNotifications(user.ID)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to retrieve notifications")
		return
	}

	RespondWithJSON(w, http.StatusOK, models.NotificationList{
		HasUnread:     hasUnread,
		Notifications: items,
	})
}

func (s *Server) handleMarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	if user == nil {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	if err := s.store.MarkAllNotificationsRead(user.ID); err != nil {
		RespondWithError(w, http.StatusInternalServerError, "Failed to mark notifications as read")
		return
	}

	RespondWithJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

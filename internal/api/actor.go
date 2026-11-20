package api

import (
	"net/http"

	"github.com/vrsandeep/mango-go/internal/store"
)

// actorFromRequest is the signed-in user for queue and subscription changes.
// Admins are unfiltered. The auth middleware has already required a session.
func actorFromRequest(r *http.Request) store.Actor {
	user := getUserFromContext(r)
	if user == nil {
		return store.Actor{}
	}
	return store.Actor{UserID: user.ID, Admin: user.Role == "admin"}
}

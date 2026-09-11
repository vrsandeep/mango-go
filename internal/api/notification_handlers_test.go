package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestNotificationHandlers(t *testing.T) {
	server, db, _ := testutil.SetupTestServer(t)
	router := server.Router()
	st := store.New(db)

	cookie := testutil.CookieForUser(t, server, "notify-api-user", "password", "user")

	folderID := int64(3)
	chapterID := int64(9)
	if _, err := st.CreateChapterDownloadedNotification("Spy x Family", "Chapter 90", &folderID, &chapterID); err != nil {
		t.Fatalf("seed notification: %v", err)
	}

	t.Run("List notifications", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/notifications", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status %d, body %s", rr.Code, rr.Body.String())
		}

		var payload models.NotificationList
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if !payload.HasUnread {
			t.Error("expected has_unread true")
		}
		if len(payload.Notifications) != 1 {
			t.Fatalf("expected 1 notification, got %d", len(payload.Notifications))
		}
		if payload.Notifications[0].SeriesTitle != "Spy x Family" {
			t.Errorf("unexpected series title %q", payload.Notifications[0].SeriesTitle)
		}
		if payload.Notifications[0].Read {
			t.Error("notification should be unread")
		}
	})

	t.Run("Unauthorized list", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/notifications", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("Mark all read", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/notifications/read", nil)
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status %d, body %s", rr.Code, rr.Body.String())
		}

		req, _ = http.NewRequest("GET", "/api/notifications", nil)
		req.AddCookie(cookie)
		rr = httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		var payload models.NotificationList
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if payload.HasUnread {
			t.Error("expected has_unread false after marking read")
		}
		if len(payload.Notifications) != 1 || !payload.Notifications[0].Read {
			t.Errorf("expected read notification still listed, got %+v", payload.Notifications)
		}
	})
}

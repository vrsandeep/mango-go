package store_test

import (
	"testing"
	"time"

	"github.com/vrsandeep/mango-go/internal/auth"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestNotificationStore(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	hash, _ := auth.HashPassword("password")
	user, err := s.CreateUser("notify-user", hash, "user")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	folderID := int64(11)
	chapterID := int64(22)

	n, err := s.CreateChapterDownloadedNotification("One Piece", "Chapter 1100", &folderID, &chapterID)
	if err != nil {
		t.Fatalf("CreateChapterDownloadedNotification failed: %v", err)
	}
	if n.Kind != models.NotificationKindChapterDownloaded {
		t.Errorf("unexpected kind %q", n.Kind)
	}
	if n.SeriesTitle != "One Piece" || n.ChapterTitle != "Chapter 1100" {
		t.Errorf("unexpected titles: %+v", n)
	}
	if n.LocalFolderID == nil || *n.LocalFolderID != folderID {
		t.Errorf("expected folder id %d, got %+v", folderID, n.LocalFolderID)
	}

	items, hasMore, err := s.ListNotifications(user.ID)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(items))
	}
	if hasMore {
		t.Error("expected hasMore false for a single notification")
	}
	if items[0].Read {
		t.Error("new notification should be unread")
	}

	hasUnread, err := s.HasUnreadNotifications(user.ID)
	if err != nil {
		t.Fatalf("HasUnreadNotifications failed: %v", err)
	}
	if !hasUnread {
		t.Error("expected unread notifications")
	}

	if err := s.MarkAllNotificationsRead(user.ID); err != nil {
		t.Fatalf("MarkAllNotificationsRead failed: %v", err)
	}

	items, _, err = s.ListNotifications(user.ID)
	if err != nil {
		t.Fatalf("ListNotifications after read failed: %v", err)
	}
	if len(items) != 1 || !items[0].Read {
		t.Errorf("expected one read notification, got %+v", items)
	}

	hasUnread, err = s.HasUnreadNotifications(user.ID)
	if err != nil {
		t.Fatalf("HasUnreadNotifications after read failed: %v", err)
	}
	if hasUnread {
		t.Error("expected no unread notifications after marking read")
	}

	other, err := s.CreateUser("notify-other", hash, "user")
	if err != nil {
		t.Fatalf("CreateUser other failed: %v", err)
	}
	hasUnread, err = s.HasUnreadNotifications(other.ID)
	if err != nil {
		t.Fatalf("HasUnreadNotifications for other user failed: %v", err)
	}
	if !hasUnread {
		t.Error("read state should be per-user")
	}
}

func TestNotificationRetention(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	hash, _ := auth.HashPassword("password")
	user, err := s.CreateUser("notify-expire", hash, "user")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	oldTime := time.Now().Add(-store.NotificationRetention - time.Hour)
	_, err = db.Exec(`
		INSERT INTO notifications (kind, series_title, chapter_title, created_at)
		VALUES (?, ?, ?, ?)`, models.NotificationKindChapterDownloaded, "Old Series", "Old Ch", oldTime)
	if err != nil {
		t.Fatalf("insert expired notification: %v", err)
	}

	_, err = s.CreateChapterDownloadedNotification("New Series", "New Ch", nil, nil)
	if err != nil {
		t.Fatalf("CreateChapterDownloadedNotification failed: %v", err)
	}

	items, hasMore, err := s.ListNotifications(user.ID)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if hasMore {
		t.Error("expected hasMore false")
	}
	if len(items) != 1 || items[0].SeriesTitle != "New Series" {
		t.Fatalf("expected only the fresh notification, got %+v", items)
	}

	deleted, err := s.DeleteExpiredNotifications()
	if err != nil {
		t.Fatalf("DeleteExpiredNotifications failed: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected to delete 1 expired notification, got %d", deleted)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM notifications").Scan(&count); err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 remaining notification, got %d", count)
	}
}

func TestListNotificationsEmpty(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	items, hasMore, err := s.ListNotifications(1)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if hasMore {
		t.Error("expected hasMore false")
	}
	if items == nil || len(items) != 0 {
		t.Errorf("expected empty slice, got %#v", items)
	}
}

func TestListNotificationsLimit(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)

	hash, _ := auth.HashPassword("password")
	user, err := s.CreateUser("notify-limit", hash, "user")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	now := time.Now()
	for i := 0; i < store.NotificationListLimit+2; i++ {
		_, err := db.Exec(`
			INSERT INTO notifications (kind, series_title, chapter_title, created_at)
			VALUES (?, ?, ?, ?)`,
			models.NotificationKindChapterDownloaded, "Series", "Ch", now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatalf("insert notification %d: %v", i, err)
		}
	}

	items, hasMore, err := s.ListNotifications(user.ID)
	if err != nil {
		t.Fatalf("ListNotifications failed: %v", err)
	}
	if !hasMore {
		t.Error("expected hasMore true when notifications exceed the limit")
	}
	if len(items) != store.NotificationListLimit {
		t.Fatalf("expected %d notifications, got %d", store.NotificationListLimit, len(items))
	}
}

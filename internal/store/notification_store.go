package store

import (
	"database/sql"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

// NotificationRetention is how long download notifications remain visible.
const NotificationRetention = 7 * 24 * time.Hour

// NotificationListLimit is the maximum number of notifications returned for the header panel.
const NotificationListLimit = 10

func notificationCutoff() time.Time {
	return time.Now().Add(-NotificationRetention)
}

// CreateChapterDownloadedNotification records that a chapter finished downloading.
func (s *Store) CreateChapterDownloadedNotification(seriesTitle, chapterTitle string, folderID, chapterID *int64) (*models.Notification, error) {
	now := time.Now()
	res, err := s.db.Exec(`
		INSERT INTO notifications (kind, series_title, chapter_title, local_folder_id, local_chapter_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		models.NotificationKindChapterDownloaded, seriesTitle, chapterTitle, folderID, chapterID, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &models.Notification{
		ID:             id,
		Kind:           models.NotificationKindChapterDownloaded,
		SeriesTitle:    seriesTitle,
		ChapterTitle:   chapterTitle,
		LocalFolderID:  folderID,
		LocalChapterID: chapterID,
		CreatedAt:      now,
		Read:           false,
	}, nil
}

// ListNotifications returns up to NotificationListLimit non-expired notifications for a user, newest first.
// hasMore is true when additional notifications exist beyond that limit.
func (s *Store) ListNotifications(userID int64) ([]*models.Notification, bool, error) {
	rows, err := s.db.Query(`
		SELECT n.id, n.kind, n.series_title, n.chapter_title, n.local_folder_id, n.local_chapter_id, n.created_at,
		       CASE WHEN r.notification_id IS NULL THEN 0 ELSE 1 END AS is_read
		FROM notifications n
		LEFT JOIN user_notification_reads r ON r.notification_id = n.id AND r.user_id = ?
		WHERE n.created_at > ?
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT ?`,
		userID, notificationCutoff(), NotificationListLimit+1,
	)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var items []*models.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, n)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(items) > NotificationListLimit
	if hasMore {
		items = items[:NotificationListLimit]
	}
	if items == nil {
		items = []*models.Notification{}
	}
	return items, hasMore, nil
}

// HasUnreadNotifications reports whether the user has any unread, non-expired notifications.
func (s *Store) HasUnreadNotifications(userID int64) (bool, error) {
	var has bool
	err := s.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM notifications n
			LEFT JOIN user_notification_reads r ON r.notification_id = n.id AND r.user_id = ?
			WHERE n.created_at > ? AND r.notification_id IS NULL
		)`, userID, notificationCutoff(),
	).Scan(&has)
	return has, err
}

// MarkAllNotificationsRead marks every current notification as read for the user.
func (s *Store) MarkAllNotificationsRead(userID int64) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO user_notification_reads (user_id, notification_id, read_at)
		SELECT ?, id, ? FROM notifications WHERE created_at > ?`,
		userID, time.Now(), notificationCutoff(),
	)
	return err
}

// DeleteExpiredNotifications removes notifications older than the retention window.
func (s *Store) DeleteExpiredNotifications() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM notifications WHERE created_at <= ?`, notificationCutoff())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanNotification(rows *sql.Rows) (*models.Notification, error) {
	var n models.Notification
	var folderID, chapterID sql.NullInt64
	if err := rows.Scan(&n.ID, &n.Kind, &n.SeriesTitle, &n.ChapterTitle, &folderID, &chapterID, &n.CreatedAt, &n.Read); err != nil {
		return nil, err
	}
	if folderID.Valid {
		n.LocalFolderID = &folderID.Int64
	}
	if chapterID.Valid {
		n.LocalChapterID = &chapterID.Int64
	}
	return &n, nil
}

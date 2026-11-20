// To handle all database interactions. This is our
// data access layer, keeping SQL queries separate from business logic.

package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

// Store provides all functions to interact with the database.
type Store struct {
	db *sql.DB
}

// New creates a new Store instance.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// AddChaptersToQueue adds multiple chapters to the download queue in a single transaction.
func (s *Store) AddChaptersToQueue(seriesTitle, providerID string, chapters []models.ChapterResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT OR IGNORE INTO download_queue
        (series_title, chapter_title, chapter_identifier, provider_id, created_at)
        VALUES (?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, ch := range chapters {
		_, err := stmt.Exec(seriesTitle, ch.Title, ch.Identifier, providerID, time.Now())
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// AddChaptersToQueueForUser records the user who queued the chapters.
func (s *Store) AddChaptersToQueueForUser(userID int64, seriesTitle, providerID string, chapters []models.ChapterResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
        INSERT OR IGNORE INTO download_queue
        (series_title, chapter_title, chapter_identifier, provider_id, created_at, user_id)
        VALUES (?, ?, ?, ?, ?, ?)
    `)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, ch := range chapters {
		if _, err := stmt.Exec(seriesTitle, ch.Title, ch.Identifier, providerID, time.Now(), userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SubscribeToSeries adds a series to the subscriptions table.
func (s *Store) SubscribeToSeries(seriesTitle, seriesIdentifier, providerID string) (*models.Subscription, error) {
	return s.SubscribeToSeriesWithFolder(seriesTitle, seriesIdentifier, providerID, nil)
}

// SubscribeToSeriesWithFolderForUser stores the user who subscribed.
// A series already subscribed by someone else is returned so the caller can refuse it.
func (s *Store) SubscribeToSeriesWithFolderForUser(userID int64, seriesTitle, seriesIdentifier, providerID string, folderPath *string) (*models.Subscription, error) {
	return s.subscribe(seriesTitle, seriesIdentifier, providerID, folderPath, &userID)
}

// SubscribeToSeriesWithFolder adds a series to the subscriptions table with a custom folder path.
func (s *Store) SubscribeToSeriesWithFolder(seriesTitle, seriesIdentifier, providerID string, folderPath *string) (*models.Subscription, error) {
	return s.subscribe(seriesTitle, seriesIdentifier, providerID, folderPath, nil)
}

func (s *Store) subscribe(seriesTitle, seriesIdentifier, providerID string, folderPath *string, userID *int64) (*models.Subscription, error) {
	var sub models.Subscription
	query := `
        INSERT INTO subscriptions (series_title, series_identifier, provider_id, folder_path, created_at, last_checked_at, user_id)
        VALUES (?, ?, ?, ?, ?, NULL, ?)
        ON CONFLICT(series_identifier, provider_id) DO NOTHING
        RETURNING id, series_title, series_identifier, provider_id, folder_path, created_at, user_id;
    `
	var owner sql.NullInt64
	err := s.db.QueryRow(query, seriesTitle, seriesIdentifier, providerID, folderPath, time.Now(), userID).Scan(
		&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &sub.FolderPath, &sub.CreatedAt, &owner,
	)
	if err == sql.ErrNoRows {
		// This means the subscription already existed, which is not an error.
		// We can fetch the existing one to return it.
		err = s.db.QueryRow(`
			SELECT id, series_title, series_identifier, provider_id, folder_path, created_at, user_id
			FROM subscriptions
        	WHERE series_identifier = ? AND provider_id = ?`, seriesIdentifier, providerID).Scan(
			&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &sub.FolderPath, &sub.CreatedAt, &owner,
		)
	}
	if owner.Valid {
		id := owner.Int64
		sub.UserID = &id
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *Store) GetDownloadQueue() ([]*models.DownloadQueueItem, error) {
	return s.GetDownloadQueueFor(Actor{Admin: true})
}

// GetDownloadQueueFor returns queue rows visible to actor.
func (s *Store) GetDownloadQueueFor(actor Actor) ([]*models.DownloadQueueItem, error) {
	clause, args := actor.owns("user_id")
	query := `
        SELECT id, series_title, chapter_title, chapter_identifier, provider_id, status, progress, message, created_at,
               local_chapter_id, local_folder_id
        FROM download_queue WHERE 1=1` + clause + ` ORDER BY created_at DESC
    `
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.DownloadQueueItem
	for rows.Next() {
		var item models.DownloadQueueItem
		var msg sql.NullString
		var localChap, localFolder sql.NullInt64
		if err := rows.Scan(&item.ID, &item.SeriesTitle, &item.ChapterTitle, &item.ChapterIdentifier, &item.ProviderID, &item.Status, &item.Progress, &msg, &item.CreatedAt, &localChap, &localFolder); err != nil {
			return nil, err
		}
		item.Message = msg.String
		if localChap.Valid {
			v := localChap.Int64
			item.LocalChapterID = &v
		}
		if localFolder.Valid {
			v := localFolder.Int64
			item.LocalFolderID = &v
		}
		items = append(items, &item)
	}
	return items, nil
}

// GetQueuedDownloadItems retrieves a limited number of items with a 'queued' status.
func (s *Store) GetQueuedDownloadItems(limit int) ([]*models.DownloadQueueItem, error) {
	query := `
        SELECT id, series_title, chapter_title, chapter_identifier, provider_id, status, progress, message, created_at,
               local_chapter_id, local_folder_id
        FROM download_queue WHERE status = 'queued' ORDER BY created_at ASC LIMIT ?
    `
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.DownloadQueueItem
	for rows.Next() {
		var item models.DownloadQueueItem
		var msg sql.NullString
		var localChap, localFolder sql.NullInt64
		if err := rows.Scan(&item.ID, &item.SeriesTitle, &item.ChapterTitle, &item.ChapterIdentifier, &item.ProviderID, &item.Status, &item.Progress, &msg, &item.CreatedAt, &localChap, &localFolder); err != nil {
			return nil, err
		}
		item.Message = msg.String
		if localChap.Valid {
			v := localChap.Int64
			item.LocalChapterID = &v
		}
		if localFolder.Valid {
			v := localFolder.Int64
			item.LocalFolderID = &v
		}
		items = append(items, &item)
	}
	return items, nil
}

// GetDownloadQueueItem retrieves a single item from the download queue by ID.
func (s *Store) GetDownloadQueueItem(id int64) (*models.DownloadQueueItem, error) {
	query := `
        SELECT id, series_title, chapter_title, chapter_identifier, provider_id, status, progress, message, created_at,
               local_chapter_id, local_folder_id
        FROM download_queue WHERE id = ?
    `
	var item models.DownloadQueueItem
	var msg sql.NullString
	var localChap, localFolder sql.NullInt64
	err := s.db.QueryRow(query, id).Scan(&item.ID, &item.SeriesTitle, &item.ChapterTitle, &item.ChapterIdentifier, &item.ProviderID, &item.Status, &item.Progress, &msg, &item.CreatedAt, &localChap, &localFolder)
	if err != nil {
		return nil, err
	}
	item.Message = msg.String
	if localChap.Valid {
		v := localChap.Int64
		item.LocalChapterID = &v
	}
	if localFolder.Valid {
		v := localFolder.Int64
		item.LocalFolderID = &v
	}
	return &item, nil
}

// UpdateDownloadQueueLocalChapter stores library IDs for a completed download (reader link).
func (s *Store) UpdateDownloadQueueLocalChapter(id, chapterID, folderID int64) error {
	_, err := s.db.Exec(`UPDATE download_queue SET local_chapter_id = ?, local_folder_id = ? WHERE id = ?`, chapterID, folderID, id)
	return err
}

// UpdateQueueItemStatus changes an item's status and message.
func (s *Store) UpdateQueueItemStatus(id int64, status, message string) error {
	query := "UPDATE download_queue SET status = ?, message = ? WHERE id = ?"
	_, err := s.db.Exec(query, status, message, id)
	return err
}

// UpdateQueueItemProgress changes an item's progress percentage.
func (s *Store) UpdateQueueItemProgress(id int64, progress int) error {
	query := "UPDATE download_queue SET progress = ? WHERE id = ?"
	_, err := s.db.Exec(query, progress, id)
	return err
}

// ResetInProgressQueueItems sets items from 'in_progress' back to 'queued' on startup.
func (s *Store) ResetInProgressQueueItems() error {
	query := "UPDATE download_queue SET status = 'queued', progress = 0, message = 'Re-queued after restart' WHERE status = 'in_progress'"
	_, err := s.db.Exec(query)
	return err
}

// PauseAllQueueItems sets all items in 'in_progress' or 'queued' to 'paused'.
func (s *Store) PauseAllQueueItems() error {
	return s.PauseAllQueueItemsFor(Actor{Admin: true})
}

// PauseAllQueueItemsFor pauses queued and in-progress rows visible to actor.
func (s *Store) PauseAllQueueItemsFor(actor Actor) error {
	clause, args := actor.owns("user_id")
	query := "UPDATE download_queue SET status = 'paused', message = 'Paused by user' WHERE (status = 'in_progress' OR status = 'queued')" + clause
	_, err := s.db.Exec(query, args...)
	return err
}

// ResumeAllQueueItems sets all items in 'paused' back to 'queued'.
func (s *Store) ResumeAllQueueItems() error {
	return s.ResumeAllQueueItemsFor(Actor{Admin: true})
}

// ResumeAllQueueItemsFor queues paused rows visible to actor.
func (s *Store) ResumeAllQueueItemsFor(actor Actor) error {
	clause, args := actor.owns("user_id")
	query := "UPDATE download_queue SET status = 'queued', message = 'Resumed by user' WHERE status = 'paused'" + clause
	_, err := s.db.Exec(query, args...)
	return err
}

// ResetFailedQueueItems sets items from 'failed' back to 'queued' to be retried.
func (s *Store) ResetFailedQueueItems() error {
	return s.ResetFailedQueueItemsFor(Actor{Admin: true})
}

// ResetFailedQueueItemsFor re-queues failed rows visible to actor.
func (s *Store) ResetFailedQueueItemsFor(actor Actor) error {
	clause, args := actor.owns("user_id")
	query := "UPDATE download_queue SET status = 'queued', progress = 0, message = 'Re-queued by user' WHERE status = 'failed'" + clause
	_, err := s.db.Exec(query, args...)
	return err
}

// DeleteCompletedQueueItems removes successfully completed items from the queue.
func (s *Store) DeleteCompletedQueueItems() error {
	return s.DeleteCompletedQueueItemsFor(Actor{Admin: true})
}

// DeleteCompletedQueueItemsFor removes completed rows visible to actor.
func (s *Store) DeleteCompletedQueueItemsFor(actor Actor) error {
	clause, args := actor.owns("user_id")
	query := "DELETE FROM download_queue WHERE status = 'completed'" + clause
	_, err := s.db.Exec(query, args...)
	return err
}

// EmptyQueue removes all items from the queue that are not completed or in progress.
func (s *Store) EmptyQueue() error {
	return s.EmptyQueueFor(Actor{Admin: true})
}

// EmptyQueueFor removes queued, failed, and paused rows visible to actor.
func (s *Store) EmptyQueueFor(actor Actor) error {
	clause, args := actor.owns("user_id")
	query := "DELETE FROM download_queue WHERE (status = 'queued' OR status = 'failed' OR status = 'paused')" + clause
	_, err := s.db.Exec(query, args...)
	return err
}

// QueueItemVisible reports whether actor may change this queue row.
func (s *Store) QueueItemVisible(actor Actor, id int64) error {
	clause, args := actor.owns("user_id")
	args = append([]any{id}, args...)
	var found int64
	err := s.db.QueryRow("SELECT id FROM download_queue WHERE id = ?"+clause, args...).Scan(&found)
	return err
}

// DeleteQueueItem removes a specific item from the download queue by ID.
func (s *Store) DeleteQueueItem(id int64) error {
	_, err := s.db.Exec("DELETE FROM download_queue WHERE id = ?", id)
	return err
}

// PauseQueueItem pauses a specific item in the download queue by ID.
func (s *Store) PauseQueueItem(id int64) error {
	result, err := s.db.Exec("UPDATE download_queue SET status = 'paused', message = 'Paused by user' WHERE id = ?", id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("download queue item with ID %d not found", id)
	}
	return nil
}

// ResumeQueueItem resumes a specific item in the download queue by ID.
func (s *Store) ResumeQueueItem(id int64) error {
	result, err := s.db.Exec("UPDATE download_queue SET status = 'queued', message = 'Resumed by user' WHERE id = ?", id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("download queue item with ID %d not found", id)
	}
	return nil
}

// RetryQueueItem retries a specific failed item in the download queue by ID.
func (s *Store) RetryQueueItem(id int64) error {
	result, err := s.db.Exec("UPDATE download_queue SET status = 'queued', progress = 0, message = 'Re-queued for retry by user' WHERE id = ? AND status = 'failed'", id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("download queue item with ID %d not found or not in failed status", id)
	}
	return nil
}

// GetAllSubscriptions retrieves all subscriptions, optionally filtered by provider ID.
func (s *Store) GetAllSubscriptions(providerIDFilter string) ([]*models.Subscription, error) {
	query := `SELECT id, series_title, series_identifier, provider_id, folder_path, created_at, last_checked_at, last_downloaded_at
		FROM subscriptions`
	args := []interface{}{}
	if providerIDFilter != "" {
		query += " WHERE provider_id = ?"
		args = append(args, providerIDFilter)
	}
	query += " ORDER BY created_at DESC, series_title ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*models.Subscription
	for rows.Next() {
		var sub models.Subscription
		var createdAt time.Time
		var lastCheckedAt, lastDownloadedAt sql.NullTime
		var folderPath sql.NullString
		if err := rows.Scan(&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &folderPath, &createdAt, &lastCheckedAt, &lastDownloadedAt); err != nil {
			return nil, err
		}
		sub.CreatedAt = createdAt
		if lastCheckedAt.Valid {
			sub.LastCheckedAt = &lastCheckedAt.Time
		}
		if lastDownloadedAt.Valid {
			sub.LastDownloadedAt = &lastDownloadedAt.Time
		}
		if folderPath.Valid {
			sub.FolderPath = &folderPath.String
		}
		subs = append(subs, &sub)
	}
	return subs, nil
}

// ListSubscriptions returns a page of subscriptions filtered by provider and search.
func (s *Store) ListSubscriptions(providerID, search string, page, perPage int, sortBy, sortDir string) ([]*models.Subscription, int, error) {
	return s.ListSubscriptionsFor(Actor{Admin: true}, providerID, search, page, perPage, sortBy, sortDir)
}

// ListSubscriptionsFor returns subscriptions visible to actor.
func (s *Store) ListSubscriptionsFor(actor Actor, providerID, search string, page, perPage int, sortBy, sortDir string) ([]*models.Subscription, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 200 {
		perPage = 200
	}

	where := []string{"1=1"}
	args := []interface{}{}
	if !actor.Admin {
		where = append(where, "user_id = ?")
		args = append(args, actor.UserID)
	}
	if providerID != "" {
		where = append(where, "provider_id = ?")
		args = append(args, providerID)
	}
	if search != "" {
		like := "%" + search + "%"
		where = append(where, "(series_title LIKE ? OR series_identifier LIKE ? OR provider_id LIKE ? OR IFNULL(folder_path, '') LIKE ?)")
		args = append(args, like, like, like, like)
	}
	whereSQL := strings.Join(where, " AND ")

	var total int
	countQuery := "SELECT COUNT(*) FROM subscriptions WHERE " + whereSQL
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderCol := "series_title"
	switch sortBy {
	case "series_title", "provider_id", "folder_path", "created_at", "last_checked_at", "last_downloaded_at":
		orderCol = sortBy
	}
	dir := "ASC"
	if strings.EqualFold(sortDir, "desc") {
		dir = "DESC"
	}

	offset := (page - 1) * perPage
	query := fmt.Sprintf(`SELECT id, series_title, series_identifier, provider_id, folder_path, created_at, last_checked_at, last_downloaded_at
		FROM subscriptions WHERE %s ORDER BY %s %s, series_title ASC LIMIT ? OFFSET ?`, whereSQL, orderCol, dir)
	queryArgs := append(append([]interface{}{}, args...), perPage, offset)

	rows, err := s.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	subs := []*models.Subscription{}
	for rows.Next() {
		var sub models.Subscription
		var createdAt time.Time
		var lastCheckedAt, lastDownloadedAt sql.NullTime
		var folderPath sql.NullString
		if err := rows.Scan(&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &folderPath, &createdAt, &lastCheckedAt, &lastDownloadedAt); err != nil {
			return nil, 0, err
		}
		sub.CreatedAt = createdAt
		if lastCheckedAt.Valid {
			sub.LastCheckedAt = &lastCheckedAt.Time
		}
		if lastDownloadedAt.Valid {
			sub.LastDownloadedAt = &lastDownloadedAt.Time
		}
		if folderPath.Valid {
			sub.FolderPath = &folderPath.String
		}
		subs = append(subs, &sub)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return subs, total, nil
}

// GetSubscriptionByID retrieves a single subscription by its primary key.
func (s *Store) GetSubscriptionByID(id int64) (*models.Subscription, error) {
	var sub models.Subscription
	var createdAt time.Time
	var lastCheckedAt, lastDownloadedAt sql.NullTime
	var folderPath sql.NullString
	query := "SELECT id, series_title, series_identifier, provider_id, folder_path, created_at, last_checked_at, last_downloaded_at FROM subscriptions WHERE id = ?"
	err := s.db.QueryRow(query, id).Scan(&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &folderPath, &createdAt, &lastCheckedAt, &lastDownloadedAt)
	if err != nil {
		return nil, err
	}
	sub.CreatedAt = createdAt
	if lastCheckedAt.Valid {
		sub.LastCheckedAt = &lastCheckedAt.Time
	}
	if lastDownloadedAt.Valid {
		sub.LastDownloadedAt = &lastDownloadedAt.Time
	}
	if folderPath.Valid {
		sub.FolderPath = &folderPath.String
	}
	return &sub, nil
}

// UpdateSubscriptionFolderPath updates the folder path for a subscription.
func (s *Store) UpdateSubscriptionFolderPath(id int64, folderPath *string) error {
	return s.UpdateSubscriptionFolderPathFor(Actor{Admin: true}, id, folderPath)
}

// UpdateSubscriptionFolderPathFor updates a subscription the actor is allowed to change.
func (s *Store) UpdateSubscriptionFolderPathFor(actor Actor, id int64, folderPath *string) error {
	clause, args := actor.owns("user_id")
	queryArgs := append([]any{folderPath, id}, args...)
	result, err := s.db.Exec("UPDATE subscriptions SET folder_path = ? WHERE id = ?"+clause, queryArgs...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("subscription with id %d not found", id)
	}

	return nil
}

// DeleteSubscription removes a subscription from the database.
func (s *Store) DeleteSubscription(id int64) error {
	return s.DeleteSubscriptionFor(Actor{Admin: true}, id)
}

// DeleteSubscriptionFor removes a subscription the actor is allowed to change.
// A missing or hidden row returns sql.ErrNoRows.
func (s *Store) DeleteSubscriptionFor(actor Actor, id int64) error {
	clause, args := actor.owns("user_id")
	queryArgs := append([]any{id}, args...)
	result, err := s.db.Exec("DELETE FROM subscriptions WHERE id = ?"+clause, queryArgs...)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SubscriptionVisible reports whether actor may recheck this subscription.
func (s *Store) SubscriptionVisible(actor Actor, id int64) error {
	clause, args := actor.owns("user_id")
	queryArgs := append([]any{id}, args...)
	var found int64
	return s.db.QueryRow("SELECT id FROM subscriptions WHERE id = ?"+clause, queryArgs...).Scan(&found)
}

// UpdateSubscriptionLastChecked sets the last_checked_at timestamp to the current time.
func (s *Store) UpdateSubscriptionLastChecked(id int64) error {
	_, err := s.db.Exec("UPDATE subscriptions SET last_checked_at = ? WHERE id = ?", time.Now(), id)
	return err
}

// UpdateSubscriptionLastDownloaded sets last_downloaded_at for the matching series and provider.
func (s *Store) UpdateSubscriptionLastDownloaded(seriesTitle, providerID string) error {
	_, err := s.db.Exec(
		"UPDATE subscriptions SET last_downloaded_at = ? WHERE series_title = ? AND provider_id = ?",
		time.Now(), seriesTitle, providerID,
	)
	return err
}

// GetChapterIdentifiersInQueue returns a slice of all chapter identifiers for a given
// series that are currently in the download queue to prevent adding duplicates.
func (s *Store) GetChapterIdentifiersInQueue(seriesTitle, providerID string) ([]string, error) {
	query := "SELECT chapter_identifier FROM download_queue WHERE series_title = ? AND provider_id = ? and status = 'queued'"
	rows, err := s.db.Query(query, seriesTitle, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identifiers []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		identifiers = append(identifiers, id)
	}
	return identifiers, nil
}

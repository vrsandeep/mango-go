// To handle all database interactions. This is our
// data access layer, keeping SQL queries separate from business logic.

package store

import (
	"database/sql"
	"fmt"
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

// SubscribeToSeries adds a series to the subscriptions table.
func (s *Store) SubscribeToSeries(seriesTitle, seriesIdentifier, providerID string) (*models.Subscription, error) {
	return s.SubscribeToSeriesWithFolder(seriesTitle, seriesIdentifier, providerID, nil)
}

// SubscribeToSeriesWithFolder adds a series to the subscriptions table with a custom folder path.
func (s *Store) SubscribeToSeriesWithFolder(seriesTitle, seriesIdentifier, providerID string, folderPath *string) (*models.Subscription, error) {
	var sub models.Subscription
	query := `
        INSERT INTO subscriptions (series_title, series_identifier, provider_id, folder_path, created_at, last_checked_at)
        VALUES (?, ?, ?, ?, ?, NULL)
        ON CONFLICT(series_identifier, provider_id) DO NOTHING
        RETURNING id, series_title, series_identifier, provider_id, folder_path, created_at;
    `
	err := s.db.QueryRow(query, seriesTitle, seriesIdentifier, providerID, folderPath, time.Now()).Scan(
		&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &sub.FolderPath, &sub.CreatedAt,
	)
	if err == sql.ErrNoRows {
		// This means the subscription already existed, which is not an error.
		// We can fetch the existing one to return it.
		err = s.db.QueryRow(`
			SELECT id, series_title, series_identifier, provider_id, folder_path, created_at
			FROM subscriptions
        	WHERE series_identifier = ? AND provider_id = ?`, seriesIdentifier, providerID).Scan(
			&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &sub.FolderPath, &sub.CreatedAt,
		)
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *Store) GetDownloadQueue() ([]*models.DownloadQueueItem, error) {
	query := `
        SELECT id, series_title, chapter_title, chapter_identifier, provider_id, status, progress, message, created_at
        FROM download_queue ORDER BY created_at DESC
    `
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.DownloadQueueItem
	for rows.Next() {
		var item models.DownloadQueueItem
		var msg sql.NullString
		if err := rows.Scan(&item.ID, &item.SeriesTitle, &item.ChapterTitle, &item.ChapterIdentifier, &item.ProviderID, &item.Status, &item.Progress, &msg, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Message = msg.String
		items = append(items, &item)
	}
	return items, nil
}

// GetQueuedDownloadItems retrieves a limited number of items with a 'queued' status.
func (s *Store) GetQueuedDownloadItems(limit int) ([]*models.DownloadQueueItem, error) {
	query := `
        SELECT id, series_title, chapter_title, chapter_identifier, provider_id, status, progress, message, created_at
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
		if err := rows.Scan(&item.ID, &item.SeriesTitle, &item.ChapterTitle, &item.ChapterIdentifier, &item.ProviderID, &item.Status, &item.Progress, &msg, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Message = msg.String
		items = append(items, &item)
	}
	return items, nil
}

// GetDownloadQueueItem retrieves a single item from the download queue by ID.
func (s *Store) GetDownloadQueueItem(id int64) (*models.DownloadQueueItem, error) {
	query := `
        SELECT id, series_title, chapter_title, chapter_identifier, provider_id, status, progress, message, created_at
        FROM download_queue WHERE id = ?
    `
	var item models.DownloadQueueItem
	var msg sql.NullString
	err := s.db.QueryRow(query, id).Scan(&item.ID, &item.SeriesTitle, &item.ChapterTitle, &item.ChapterIdentifier, &item.ProviderID, &item.Status, &item.Progress, &msg, &item.CreatedAt)
	if err != nil {
		return nil, err
	}
	item.Message = msg.String
	return &item, nil
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
	query := "UPDATE download_queue SET status = 'paused', message = 'Paused by user' where status = 'in_progress' OR status = 'queued'"
	_, err := s.db.Exec(query)
	return err
}

// ResumeAllQueueItems sets all items in 'paused' back to 'queued'.
func (s *Store) ResumeAllQueueItems() error {
	query := "UPDATE download_queue SET status = 'queued', message = 'Resumed by user' WHERE status = 'paused'"
	_, err := s.db.Exec(query)
	return err
}

// ResetFailedQueueItems sets items from 'failed' back to 'queued' to be retried.
func (s *Store) ResetFailedQueueItems() error {
	query := "UPDATE download_queue SET status = 'queued', progress = 0, message = 'Re-queued by user' WHERE status = 'failed'"
	_, err := s.db.Exec(query)
	return err
}

// DeleteCompletedQueueItems removes successfully completed items from the queue.
func (s *Store) DeleteCompletedQueueItems() error {
	query := "DELETE FROM download_queue WHERE status = 'completed'"
	_, err := s.db.Exec(query)
	return err
}

// EmptyQueue removes all items from the queue that are not completed or in progress.
func (s *Store) EmptyQueue() error {
	query := "DELETE FROM download_queue WHERE status = 'queued' OR status = 'failed' OR status = 'paused'"
	_, err := s.db.Exec(query)
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
	query := "SELECT id, series_title, series_identifier, provider_id, folder_path, created_at, last_checked_at FROM subscriptions"
	args := []interface{}{}
	if providerIDFilter != "" {
		query += " WHERE provider_id = ?"
		args = append(args, providerIDFilter)
	}
	query += " ORDER BY series_title ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*models.Subscription
	for rows.Next() {
		var sub models.Subscription
		var createdAt time.Time
		var lastCheckedAt sql.NullTime
		var folderPath sql.NullString
		if err := rows.Scan(&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &folderPath, &createdAt, &lastCheckedAt); err != nil {
			return nil, err
		}
		sub.CreatedAt = createdAt
		if lastCheckedAt.Valid {
			sub.LastCheckedAt = &lastCheckedAt.Time
		}
		if folderPath.Valid {
			sub.FolderPath = &folderPath.String
		}
		subs = append(subs, &sub)
	}
	return subs, nil
}

// GetSubscriptionByID retrieves a single subscription by its primary key.
func (s *Store) GetSubscriptionByID(id int64) (*models.Subscription, error) {
	var sub models.Subscription
	var createdAt time.Time
	var lastCheckedAt sql.NullTime
	var folderPath sql.NullString
	query := "SELECT id, series_title, series_identifier, provider_id, folder_path, created_at, last_checked_at FROM subscriptions WHERE id = ?"
	err := s.db.QueryRow(query, id).Scan(&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &folderPath, &createdAt, &lastCheckedAt)
	if err != nil {
		return nil, err
	}
	sub.CreatedAt = createdAt
	if lastCheckedAt.Valid {
		sub.LastCheckedAt = &lastCheckedAt.Time
	}
	if folderPath.Valid {
		sub.FolderPath = &folderPath.String
	}
	return &sub, nil
}

// UpdateSubscriptionFolderPath updates the folder path for a subscription.
func (s *Store) UpdateSubscriptionFolderPath(id int64, folderPath *string) error {
	result, err := s.db.Exec("UPDATE subscriptions SET folder_path = ? WHERE id = ?", folderPath, id)
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
	_, err := s.db.Exec("DELETE FROM subscriptions WHERE id = ?", id)
	return err
}

// UpdateSubscriptionLastChecked sets the last_checked_at timestamp to the current time.
func (s *Store) UpdateSubscriptionLastChecked(id int64) error {
	_, err := s.db.Exec("UPDATE subscriptions SET last_checked_at = ? WHERE id = ?", time.Now(), id)
	return err
}

// GetChapterIdentifiersInQueue returns a slice of all chapter identifiers for a given
// series that are currently in the download queue to prevent adding duplicates.
func (s *Store) GetChapterIdentifiersInQueue(seriesTitle, providerID string) ([]string, error) {
	query := "SELECT chapter_identifier FROM download_queue WHERE series_title = ? AND provider_id = ?"
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

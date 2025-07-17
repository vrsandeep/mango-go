// To handle all database interactions. This is our
// data access layer, keeping SQL queries separate from business logic.

package store

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/util"
)

// Store provides all functions to interact with the database.
type Store struct {
	db *sql.DB
}

// New creates a new Store instance.
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetOrCreateSeries finds a series by title or creates it if it doesn't exist.
// It returns the ID of the series. This operation must be done in a transaction.
func (s *Store) GetOrCreateSeries(tx *sql.Tx, title, path string) (int64, error) {
	var seriesID int64
	// Try to find the series first.
	err := tx.QueryRow("SELECT id FROM series WHERE title = ?", title).Scan(&seriesID)
	if err == sql.ErrNoRows {
		// Series does not exist, so create it.
		res, err := tx.Exec("INSERT INTO series (title, path, created_at, updated_at) VALUES (?, ?, ?, ?)",
			title, path, time.Now(), time.Now())
		if err != nil {
			return 0, err
		}
		seriesID, err = res.LastInsertId()
		if err != nil {
			return 0, err
		}
	} else if err != nil {
		// Another error occurred.
		return 0, err
	}
	return seriesID, nil
}

// UpdateSeriesCoverURL updates the custom cover URL for a given series.
func (s *Store) UpdateSeriesCoverURL(seriesID int64, url string) (int64, error) {
	rows, err := s.db.Exec("UPDATE series SET custom_cover_url = ? WHERE id = ?", url, seriesID)
	affected, _ := rows.RowsAffected()
	return affected, err
}

// MarkAllChaptersAs updates the 'read' status for all chapters of a series.
func (s *Store) MarkAllChaptersAs(seriesID int64, read bool, userID int64) error {
	// get chapter ids from series by joining chapters and series tables
	query := `
		SELECT c.id
		FROM chapters c
		JOIN series s ON c.series_id = s.id
		WHERE s.id = ?
	`
	rows, err := s.db.Query(query, seriesID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var chapterIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		chapterIDs = append(chapterIDs, id)
	}

	// Update user_chapter_progress for each chapter individually
	query = `
		INSERT INTO user_chapter_progress (user_id, chapter_id, progress_percent, read, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, chapter_id) DO UPDATE SET
			progress_percent = excluded.progress_percent,
			read = excluded.read,
			updated_at = CURRENT_TIMESTAMP;
	`
	stmt, err := s.db.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	progressPercent := map[bool]int{true: 100, false: 0}[read] // Set progress to 100 if read, 0 if unread

	for _, chapterID := range chapterIDs {
		_, err := stmt.Exec(userID, chapterID, progressPercent, read)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateChapterProgress updates the reading progress for a given chapter.
func (s *Store) UpdateChapterProgress(chapterID int64, userID int64, progressPercent int, read bool) error {
	query := `
		INSERT INTO user_chapter_progress (user_id, chapter_id, progress_percent, read, updated_at)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, chapter_id) DO UPDATE SET
			progress_percent = excluded.progress_percent,
			read = excluded.read,
			updated_at = CURRENT_TIMESTAMP;
	`
	_, err := s.db.Exec(query, userID, chapterID, progressPercent, read)
	return err
}

// AddOrUpdateChapter adds a chapter or updates its page count if it already exists.
// It uses the file path as a unique identifier for the chapter.
// This operation must be done in a transaction.
func (s *Store) AddOrUpdateChapter(tx *sql.Tx, seriesID int64, path string, pageCount int, thumbnail string) (int64, error) {
	var chapterID int64
	err := tx.QueryRow("SELECT id FROM chapters WHERE path = ?", path).Scan(&chapterID)
	if err == sql.ErrNoRows {
		// Chapter does not exist, insert it.
		res, err := tx.Exec("INSERT INTO chapters (series_id, path, page_count, thumbnail, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
			seriesID, path, pageCount, thumbnail, time.Now(), time.Now())
		if err != nil {
			return 0, err
		}
		chapterID, _ = res.LastInsertId()
	} else if err != nil {
		return 0, err
	} else {
		// Chapter exists, update it.
		_, err := tx.Exec("UPDATE chapters SET page_count = ?, thumbnail = ?, updated_at = ? WHERE id = ?",
			pageCount, thumbnail, time.Now(), chapterID)
		if err != nil {
			return 0, err
		}
	}
	return chapterID, nil
}

// UpdateSeriesThumbnailIfNeeded sets the series thumbnail only if it's not already set.
// This ensures the first scanned chapter's cover becomes the series cover.
func (s *Store) UpdateSeriesThumbnailIfNeeded(tx *sql.Tx, seriesID int64, thumbnail string) error {
	var currentThumbnail sql.NullString
	err := tx.QueryRow("SELECT thumbnail FROM series WHERE id = ?", seriesID).Scan(&currentThumbnail)
	if err != nil {
		return err
	}

	if !currentThumbnail.Valid || currentThumbnail.String == "" {
		_, err := tx.Exec("UPDATE series SET thumbnail = ? WHERE id = ?", thumbnail, seriesID)
		return err
	}
	return nil
}

func (s *Store) AddTagToSeries(seriesID int64, tagName string) (*models.Tag, error) {
	tagName = strings.TrimSpace(strings.ToLower(tagName))
	if tagName == "" {
		return nil, fmt.Errorf("tag name cannot be empty")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var tagID int64
	var tag *models.Tag
	tag, err = s.GetOrCreateTag(tagName, true)
	if err != nil {
		return nil, err
	}
	tagID = tag.ID

	_, err = tx.Exec("INSERT OR IGNORE INTO series_tags (series_id, tag_id) VALUES (?, ?)", seriesID, tagID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			// If the tag is already associated with the series, ignore the error
			return &models.Tag{ID: tagID, Name: tagName}, nil
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &models.Tag{ID: tagID, Name: tagName}, nil
}

func (s *Store) RemoveTagFromSeries(seriesID, tagID int64) error {
	_, err := s.db.Exec("DELETE FROM series_tags WHERE series_id = ? AND tag_id = ?", seriesID, tagID)
	if err != nil {
		return fmt.Errorf("failed to remove tag from series: %w", err)
	}
	// Check if the tag is no longer associated with any series
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM series_tags WHERE tag_id = ?", tagID).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check tag associations: %w", err)
	}
	if count == 0 {
		// If no series are left with this tag, delete the tag
		_, err = s.db.Exec("DELETE FROM tags WHERE id = ?", tagID)
		if err != nil {
			return fmt.Errorf("failed to delete tag: %w", err)
		}
	}
	return nil
}

// DeleteChapterByPath removes a chapter from the database using its file path.
func (s *Store) DeleteChapterByPath(path string) error {
	_, err := s.db.Exec("DELETE FROM chapters WHERE path = ?", path)
	if err != nil {
		log.Printf("Error deleting chapter by path %s: %v", path, err)
	}
	return err
}

// DeleteEmptySeries removes any series that have no associated chapters.
func (s *Store) DeleteEmptySeries() error {
	query := `
        DELETE FROM series
        WHERE id IN (
            SELECT s.id FROM series s
            LEFT JOIN chapters c ON s.id = c.series_id
            GROUP BY s.id
            HAVING COUNT(c.id) = 0
        )
    `
	_, err := s.db.Exec(query)
	if err != nil {
		log.Printf("Error deleting empty series: %v", err)
	}
	return err
}

// UpdateChapterThumbnail updates the thumbnail for a single chapter.
func (s *Store) UpdateChapterThumbnail(chapterID int64, thumbnail string) error {
	_, err := s.db.Exec("UPDATE chapters SET thumbnail = ? WHERE id = ?", thumbnail, chapterID)
	return err
}

// UpdateAllSeriesThumbnails iterates through all series and sets their thumbnail
// to be the same as their first chapter's thumbnail.
func (s *Store) UpdateAllSeriesThumbnails() error {
	// Get all series IDs
	rows, err := s.db.Query("SELECT id FROM series")
	if err != nil {
		return err
	}
	defer rows.Close()

	var seriesIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		seriesIDs = append(seriesIDs, id)
	}
	if rows.Err() != nil {
		return rows.Err()
	}

	// For each series, find the first chapter and update the series thumbnail
	for _, seriesID := range seriesIDs {
		// Get all chapter paths for this series
		chapterRows, err := s.db.Query("SELECT path, thumbnail FROM chapters WHERE series_id = ?", seriesID)
		if err != nil {
			log.Printf("Error getting chapters for series %d: %v", seriesID, err)
			continue
		}

		var chapters []struct {
			Path      string
			Thumbnail sql.NullString
		}
		for chapterRows.Next() {
			var c struct {
				Path      string
				Thumbnail sql.NullString
			}
			if err := chapterRows.Scan(&c.Path, &c.Thumbnail); err != nil {
				log.Printf("Error scanning chapter for series %d: %v", seriesID, err)
				continue
			}
			chapters = append(chapters, c)
		}
		chapterRows.Close()

		if len(chapters) > 0 {
			// Sort chapters naturally to find the "first" one
			sort.Slice(chapters, func(i, j int) bool {
				return util.NaturalSortLess(chapters[i].Path, chapters[j].Path)
			})

			// Use the first chapter's thumbnail for the series
			firstChapterThumbnail := chapters[0].Thumbnail
			if firstChapterThumbnail.Valid {
				_, err := s.db.Exec("UPDATE series SET thumbnail = ? WHERE id = ?", firstChapterThumbnail.String, seriesID)
				if err != nil {
					log.Printf("Error updating series thumbnail for series %d: %v", seriesID, err)
				}
			}
		}
	}
	return nil
}

// GetSeriesSettings retrieves the sort settings for a series.
func (s *Store) GetSeriesSettings(seriesID int64, userID int64) (*models.SeriesSettings, error) {
	var settings models.SeriesSettings
	err := s.db.QueryRow(`
		SELECT sort_by, sort_dir
		FROM user_series_settings
		WHERE series_id = ? AND user_id = ?
	`, seriesID, userID).Scan(&settings.SortBy, &settings.SortDir)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return default settings if not found
			settings.SortBy = "auto"
			settings.SortDir = "asc"
			return &settings, nil
		}
		return nil, err
	}
	return &settings, nil
}

// UpdateSeriesSettings saves the sort settings for a series.
func (s *Store) UpdateSeriesSettings(seriesID int64, userID int64, sortBy, sortDir string) error {
	query := `INSERT INTO user_series_settings (series_id, user_id, sort_by, sort_dir) VALUES (?, ?, ?, ?)
              ON CONFLICT(user_id, series_id) DO UPDATE SET sort_by=excluded.sort_by, sort_dir=excluded.sort_dir;`
	_, err := s.db.Exec(query, seriesID, userID, sortBy, sortDir)
	return err
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
	var sub models.Subscription
	query := `
        INSERT INTO subscriptions (series_title, series_identifier, provider_id, created_at, last_checked_at)
        VALUES (?, ?, ?, ?, NULL)
        ON CONFLICT(series_identifier, provider_id) DO NOTHING
        RETURNING id, series_title, series_identifier, provider_id, created_at;
    `
	err := s.db.QueryRow(query, seriesTitle, seriesIdentifier, providerID, time.Now()).Scan(
		&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &sub.CreatedAt,
	)
	if err == sql.ErrNoRows {
		// This means the subscription already existed, which is not an error.
		// We can fetch the existing one to return it.
		err = s.db.QueryRow(`
			SELECT id, series_title, series_identifier, provider_id, created_at
			FROM subscriptions
        	WHERE series_identifier = ? AND provider_id = ?`, seriesIdentifier, providerID).Scan(
			&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &sub.CreatedAt,
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

// GetAllSubscriptions retrieves all subscriptions, optionally filtered by provider ID.
func (s *Store) GetAllSubscriptions(providerIDFilter string) ([]*models.Subscription, error) {
	query := "SELECT id, series_title, series_identifier, provider_id, created_at, last_checked_at FROM subscriptions"
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
		if err := rows.Scan(&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &createdAt, &lastCheckedAt); err != nil {
			return nil, err
		}
		sub.CreatedAt = createdAt
		if lastCheckedAt.Valid {
			sub.LastCheckedAt = &lastCheckedAt.Time
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
	query := "SELECT id, series_title, series_identifier, provider_id, created_at, last_checked_at FROM subscriptions WHERE id = ?"
	err := s.db.QueryRow(query, id).Scan(&sub.ID, &sub.SeriesTitle, &sub.SeriesIdentifier, &sub.ProviderID, &createdAt, &lastCheckedAt)
	if err != nil {
		return nil, err
	}
	sub.CreatedAt = createdAt
	if lastCheckedAt.Valid {
		sub.LastCheckedAt = &lastCheckedAt.Time
	}
	return &sub, nil
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

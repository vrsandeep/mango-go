// To handle all database interactions. This is our
// data access layer, keeping SQL queries separate from business logic.

package store

import (
	"database/sql"
	"time"
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
func (s *Store) UpdateSeriesCoverURL(seriesID int64, url string) error {
	_, err := s.db.Exec("UPDATE series SET custom_cover_url = ? WHERE id = ?", url, seriesID)
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

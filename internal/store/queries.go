// A NEW file in the store package dedicated to read queries for the API.
// This keeps the original store.go focused on write/update operations.

package store

import (
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

// ListSeries fetches all series from the database.
func (s *Store) ListSeries() ([]*models.Series, error) {
	rows, err := s.db.Query("SELECT id, title, path, created_at, updated_at FROM series ORDER BY title")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var seriesList []*models.Series
	for rows.Next() {
		var series models.Series
		if err := rows.Scan(&series.ID, &series.Title, &series.Path, &series.CreatedAt, &series.UpdatedAt); err != nil {
			return nil, err
		}
		seriesList = append(seriesList, &series)
	}
	return seriesList, nil
}

// GetSeriesByID fetches a single series and all its associated chapters.
func (s *Store) GetSeriesByID(id int64) (*models.Series, error) {
	var series models.Series
	err := s.db.QueryRow("SELECT id, title, path FROM series WHERE id = ?", id).Scan(&series.ID, &series.Title, &series.Path)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query("SELECT id, path, page_count, read, progress_percent FROM chapters WHERE series_id = ? ORDER BY path", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var chapter models.Chapter
		if err := rows.Scan(&chapter.ID, &chapter.Path, &chapter.PageCount, &chapter.Read, &chapter.ProgressPercent); err != nil {
			return nil, err
		}
		chapter.SeriesID = id
		series.Chapters = append(series.Chapters, &chapter)
	}
	return &series, nil
}

// GetChapterByID fetches a single chapter by its ID.
func (s *Store) GetChapterByID(id int64) (*models.Chapter, error) {
	var chapter models.Chapter
	err := s.db.QueryRow("SELECT id, series_id, path, page_count, read, progress_percent FROM chapters WHERE id = ?", id).Scan(&chapter.ID, &chapter.SeriesID, &chapter.Path, &chapter.PageCount, &chapter.Read, &chapter.ProgressPercent)
	if err != nil {
		return nil, err
	}
	return &chapter, nil
}

// UpdateChapterProgress updates the reading progress for a given chapter.
func (s *Store) UpdateChapterProgress(chapterID int64, progressPercent int, read bool) error {
	_, err := s.db.Exec("UPDATE chapters SET progress_percent = ?, read = ?, updated_at = ? WHERE id = ?",
		progressPercent, read, time.Now(), chapterID)
	return err
}

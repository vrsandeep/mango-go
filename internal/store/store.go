// To handle all database interactions. This is our
// data access layer, keeping SQL queries separate from business logic.

package store

import (
	"database/sql"
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
func (s *Store) MarkAllChaptersAs(seriesID int64, read bool) error {
	_, err := s.db.Exec("UPDATE chapters SET read = ?, progress_percent = ? WHERE series_id = ?",
		read,
		map[bool]int{true: 100, false: 0}[read], // Set progress to 100 if read, 0 if unread
		seriesID)
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
	err = tx.QueryRow("SELECT id FROM tags WHERE name = ?", tagName).Scan(&tagID)
	if err == sql.ErrNoRows {
		res, err := tx.Exec("INSERT INTO tags (name) VALUES (?)", tagName)
		if err != nil {
			return nil, err
		}
		tagID, err = res.LastInsertId()
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

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

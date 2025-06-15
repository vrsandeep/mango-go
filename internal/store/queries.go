// A NEW file in the store package dedicated to read queries for the API.
// This keeps the original store.go focused on write/update operations.

package store

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/util"
)

// ListSeries fetches all series from the database.
func (s *Store) ListSeries(page, perPage int, search, sortBy, sortDir string) ([]*models.Series, int, error) {
	var args []interface{}
	var countArgs []interface{}

	// --- Count Query ---
	countQuery := "SELECT COUNT(DISTINCT s.id) FROM series s"
	if search != "" {
		countQuery += " WHERE s.title LIKE ?"
		countArgs = append(countArgs, "%"+search+"%")
	}
	var totalCount int
	if err := s.db.QueryRow(countQuery, countArgs...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// --- Main Query ---
	offset := (page - 1) * perPage
	query := `
        SELECT
            s.id, s.title, s.path, s.thumbnail, s.custom_cover_url, s.created_at, s.updated_at,
            COUNT(c.id) as total_chapters,
            SUM(CASE WHEN c.read = 1 THEN 1 ELSE 0 END) as read_chapters
        FROM series s
        LEFT JOIN chapters c ON s.id = c.series_id
    `
	if search != "" {
		query += " WHERE s.title LIKE ?"
		args = append(args, "%"+search+"%")
	}
	query += " GROUP BY s.id"

	// Sorting
	sortDir = strings.ToUpper(sortDir)
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "ASC"
	}
	switch sortBy {
	case "updated_at":
		query += fmt.Sprintf(" ORDER BY s.updated_at %s", sortDir)
	case "progress":
		query += fmt.Sprintf(" ORDER BY CAST(read_chapters AS REAL) / total_chapters %s, s.title ASC", sortDir)
	default:
		query += fmt.Sprintf(" ORDER BY s.title %s", sortDir)
	}

	query += " LIMIT ? OFFSET ?"
	args = append(args, perPage, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var seriesList []*models.Series
	for rows.Next() {
		var series models.Series
		var thumb, customCover sql.NullString
		if err := rows.Scan(
			&series.ID, &series.Title, &series.Path, &thumb, &customCover, &series.CreatedAt, &series.UpdatedAt,
			&series.TotalChapters, &series.ReadChapters,
		); err != nil {
			return nil, 0, err
		}
		series.Thumbnail = thumb.String
		series.CustomCoverURL = customCover.String
		seriesList = append(seriesList, &series)
	}
	return seriesList, totalCount, nil
}

// GetSeriesByID fetches a single series and all its associated chapters.
func (s *Store) GetSeriesByID(id int64, page, perPage int, search, sortBy, sortDir string) (*models.Series, int, error) {
	var series models.Series
	var thumb, customCover sql.NullString
	err := s.db.QueryRow("SELECT id, title, path, thumbnail, custom_cover_url FROM series WHERE id = ?", id).Scan(&series.ID, &series.Title, &series.Path, &thumb, &customCover)
	if err != nil {
		return nil, 0, err
	}
	series.Thumbnail = thumb.String
	series.CustomCoverURL = customCover.String
	series.Tags, _ = s.getTagsForSeries(id)

	// Fetch total chapters count
	var chapterArgs []interface{}
	chapterArgs = append(chapterArgs, id)
	chapterCountQuery := "SELECT COUNT(id) FROM chapters WHERE series_id = ?"
	if search != "" {
		chapterCountQuery += " AND path LIKE ?"
		chapterArgs = append(chapterArgs, "%"+search+"%")
	}
	var totalChapters int
	s.db.QueryRow(chapterCountQuery, chapterArgs...).Scan(&totalChapters)

	// Main query to fetch chapters
	chapterQuery := "SELECT id, path, page_count, read, progress_percent, thumbnail FROM chapters WHERE series_id = ?"
	if search != "" {
		chapterQuery += " AND path LIKE ?"
	}
	sortDir = strings.ToUpper(sortDir)
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "ASC"
	}
	switch sortBy {
	case "path":
		chapterQuery += fmt.Sprintf(" ORDER BY path %s", sortDir)
	case "auto":
		// Auto sort is handled in Go after fetching
		chapterQuery += " ORDER BY path ASC"
	default:
		chapterQuery += " ORDER BY path ASC"
	}

	chapterQuery += " LIMIT ? OFFSET ?"
	offset := (page - 1) * perPage
	chapterArgs = append(chapterArgs, perPage, offset)

	rows, err := s.db.Query(chapterQuery, chapterArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var chapter models.Chapter
		var chapThumb sql.NullString
		if err := rows.Scan(&chapter.ID, &chapter.Path, &chapter.PageCount, &chapter.Read, &chapter.ProgressPercent, &chapThumb); err != nil {
			return nil, 0, err
		}
		chapter.Thumbnail = chapThumb.String
		chapter.SeriesID = id
		series.Chapters = append(series.Chapters, &chapter)
	}
	if sortBy == "auto" {
		sort.Slice(series.Chapters, func(i, j int) bool {
			isLess := util.NaturalSortLess(series.Chapters[i].Path, series.Chapters[j].Path)
			if strings.ToLower(sortDir) == "desc" {
				return !isLess
			}
			return isLess
		})
	}
	return &series, totalChapters, nil
}

// GetChapterByID fetches a single chapter by its ID.
func (s *Store) GetChapterByID(id int64) (*models.Chapter, error) {
	var chapter models.Chapter
	var thumb sql.NullString
	err := s.db.QueryRow("SELECT id, series_id, path, page_count, read, progress_percent, thumbnail FROM chapters WHERE id = ?", id).Scan(&chapter.ID, &chapter.SeriesID, &chapter.Path, &chapter.PageCount, &chapter.Read, &chapter.ProgressPercent, &thumb)
	if err != nil {
		return nil, err
	}
	chapter.Thumbnail = thumb.String
	return &chapter, nil
}

// UpdateChapterProgress updates the reading progress for a given chapter.
func (s *Store) UpdateChapterProgress(chapterID int64, progressPercent int, read bool) error {
	_, err := s.db.Exec("UPDATE chapters SET progress_percent = ?, read = ?, updated_at = ? WHERE id = ?",
		progressPercent, read, time.Now(), chapterID)
	return err
}

func (s *Store) getTagsForSeries(seriesID int64) ([]*models.Tag, error) {
	query := "SELECT t.id, t.name FROM tags t JOIN series_tags st ON t.id = st.tag_id WHERE st.series_id = ?"
	rows, err := s.db.Query(query, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}
	return tags, nil
}

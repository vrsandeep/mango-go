// A NEW file in the store package dedicated to read queries for the API.
// This keeps the original store.go focused on write/update operations.

package store

import (
	"database/sql"
	"fmt"
	"slices"
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
            total_chapters, read_chapters
        FROM series s
        LEFT JOIN chapters c ON s.id = c.series_id
    `
	if search != "" {
		query += " WHERE s.title LIKE ?"
		args = append(args, "%"+search+"%")
	}

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
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			// Log the error but do not return it, as we are already returning from the function
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

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
	err := s.db.QueryRow("SELECT id, title, path, thumbnail, custom_cover_url, total_chapters FROM series WHERE id = ?", id).Scan(&series.ID, &series.Title, &series.Path, &thumb, &customCover, &series.TotalChapters)
	if err != nil {
		return nil, 0, err
	}
	series.Thumbnail = thumb.String
	series.CustomCoverURL = customCover.String
	series.Tags, _ = s.getTagsForSeries(id)

	// Fetch total chapters count
	totalChapters := series.TotalChapters
	if search != "" {
		var chapterArgs []interface{}
		chapterArgs = append(chapterArgs, id)
		chapterCountQuery := "SELECT COUNT(id) FROM chapters WHERE series_id = ?"
		chapterCountQuery += " AND path LIKE ?"
		chapterArgs = append(chapterArgs, "%"+search+"%")
		err = s.db.QueryRow(chapterCountQuery, chapterArgs...).Scan(&totalChapters)
		if err != nil {
			return &series, 0, err
		}
	}

	// Main query to fetch chapters
	chapters, err := s.getChaptersForSeries(id, page, perPage, search, sortBy, sortDir)
	series.Chapters = chapters
	if err != nil {
		return &series, 0, err
	}
	return &series, totalChapters, nil
}

func (s *Store) getChaptersForSeries(seriesId int64, page, perPage int, search, sortBy, sortDir string) ([]*models.Chapter, error) {
	chapterQuery := "SELECT id, path, page_count, read, progress_percent, thumbnail FROM chapters WHERE series_id = ?"
	var chapterArgs []interface{}
	chapterArgs = append(chapterArgs, seriesId)
	if search != "" {
		chapterQuery += " AND path LIKE ?"
		chapterArgs = append(chapterArgs, "%"+search+"%")
	}
	sortDir = strings.ToUpper(sortDir)
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "ASC"
	}
	switch sortBy {
	case "path":
		chapterQuery += fmt.Sprintf(" ORDER BY path %s LIMIT ? OFFSET ?", sortDir)
	case "auto":
		// Auto sort is handled in Go after fetching
		chapterQuery += " ORDER BY path ASC"
	default:
		chapterQuery += " ORDER BY path ASC LIMIT ? OFFSET ?"
	}

	offset := (page - 1) * perPage
	chapterArgs = append(chapterArgs, perPage, offset)
	var chapters []*models.Chapter
	rows, err := s.db.Query(chapterQuery, chapterArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var chapter models.Chapter
		var chapThumb sql.NullString
		if err := rows.Scan(&chapter.ID, &chapter.Path, &chapter.PageCount, &chapter.Read, &chapter.ProgressPercent, &chapThumb); err != nil {
			return nil, err
		}
		chapter.Thumbnail = chapThumb.String
		chapter.SeriesID = seriesId
		chapters = append(chapters, &chapter)
	}
	if sortBy == "auto" {
		// sort.Slice(series.Chapters, func(i, j int) bool {
		// 	isLess := util.NaturalSortLess(series.Chapters[i].Path, series.Chapters[j].Path)
		// 	if strings.ToLower(sortDir) == "desc" {
		// 		return !isLess
		// 	}
		// 	return isLess
		// })

		// Use the ChapterSorter to sort chapters
		chapterTitles := make([]string, len(chapters))
		for i, chapter := range chapters {
			chapterTitles[i] = getChapterTitle(chapter)
		}
		cs := util.NewChapterSorter(chapterTitles)
		slices.SortFunc(chapters, func(a, b *models.Chapter) int {
			comparison := cs.Compare(getChapterTitle(a), getChapterTitle(b))
			if strings.ToLower(sortDir) == "desc" {
				return -comparison
			}
			return comparison
		})
	}
	if sortBy == "auto" {
		// For auto sort, we send only perPage, with offset applied manually
		var newChapters []*models.Chapter
		if offset < len(chapters) {
			end := offset + perPage
			if end > len(chapters) {
				end = len(chapters)
			}
			newChapters = chapters[offset:end]
		} else {
			// If offset is beyond the length of chapters, return an empty slice
			newChapters = []*models.Chapter{}
		}
		return newChapters, nil
	}

	return chapters, nil

}

func getChapterTitle(chapter *models.Chapter) string {
	// Extract the last part of the path as the title
	parts := strings.Split(chapter.Path, "/")
	if len(parts) == 0 {
		return ""
	}
	title := parts[len(parts)-1]
	// Remove file extension if present
	if dotIndex := strings.LastIndex(title, "."); dotIndex != -1 {
		title = title[:dotIndex]
	}
	return title
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

// GetAllChapterPaths returns a slice of all chapter file paths in the DB.
func (s *Store) GetAllChapterPaths() ([]string, error) {
	rows, err := s.db.Query("SELECT path FROM chapters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

// GetAllChaptersForThumbnailing returns a slice of all chapters with just
// the ID and path, which is all that's needed for the job.
func (s *Store) GetAllChaptersForThumbnailing() ([]*models.Chapter, error) {
	rows, err := s.db.Query("SELECT id, path FROM chapters")
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			// Log the error but do not return it, as we are already returning from the function
			fmt.Printf("Error closing rows: %v\n", err)
		}
	}(rows)

	var chapters []*models.Chapter
	for rows.Next() {
		var chapter models.Chapter
		if err := rows.Scan(&chapter.ID, &chapter.Path); err != nil {
			return nil, err
		}
		chapters = append(chapters, &chapter)
	}
	return chapters, nil
}

// GetChapterNeighbors finds the previous and next chapter IDs based on sort settings.
func (s *Store) GetChapterNeighbors(seriesID, currentChapterID int64) (map[string]*int64, error) {
	settings, err := s.GetSeriesSettings(seriesID)
	if err != nil {
		return nil, err
	}
	var chapters []*models.Chapter
	chapters, err = s.getChaptersForSeries(seriesID, 1, 10000, "", settings.SortBy, settings.SortDir)
	if err != nil {
		return nil, err
	}

	// Find the index of the current chapter
	currentIndex := -1
	for i, ch := range chapters {
		if ch.ID == currentChapterID {
			currentIndex = i
			break
		}
	}

	if currentIndex == -1 {
		return map[string]*int64{"prev": nil, "next": nil}, nil
	}

	neighbors := make(map[string]*int64)
	if currentIndex > 0 {
		prevID := chapters[currentIndex-1].ID
		neighbors["prev"] = &prevID
	} else {
		neighbors["prev"] = nil
	}
	if currentIndex < len(chapters)-1 {
		nextID := chapters[currentIndex+1].ID
		neighbors["next"] = &nextID
	} else {
		neighbors["next"] = nil
	}

	return neighbors, nil
}

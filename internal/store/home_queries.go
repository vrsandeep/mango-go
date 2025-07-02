package store

import (
	"database/sql"
	"log"
	"sort"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

// GetContinueReading fetches chapters the user has started but not finished.
func (s *Store) GetContinueReading(userID int64, limit int) ([]*models.HomeSectionItem, error) {
	query := `
		SELECT
			s.id, s.title, c.id, c.path,
			COALESCE(s.custom_cover_url, s.thumbnail, '') as cover_art,
			ucp.progress_percent, ucp.read, ucp.updated_at
		FROM user_chapter_progress ucp
		JOIN chapters c ON ucp.chapter_id = c.id
		JOIN series s ON c.series_id = s.id
		WHERE ucp.user_id = ? AND ucp.read = 0 AND ucp.progress_percent > 0
		ORDER BY ucp.updated_at DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.HomeSectionItem
	for rows.Next() {
		var item models.HomeSectionItem
		var chapterID sql.NullInt64
		var progress sql.NullInt64
		var read sql.NullBool
		if err := rows.Scan(&item.SeriesID, &item.SeriesTitle, &chapterID, &item.ChapterTitle, &item.CoverArt, &progress, &read, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if chapterID.Valid {
			item.ChapterID = &chapterID.Int64
		}
		if progress.Valid {
			p := int(progress.Int64)
			item.ProgressPercent = &p
		}
		if read.Valid {
			item.Read = &read.Bool
		}
		items = append(items, &item)
	}
	return items, nil
}

// GetNextUp fetches the next unread chapter in series the user is actively reading.
func (s *Store) GetNextUp(userID int64, limit int) ([]*models.HomeSectionItem, error) {
	// Step 1: Find the most recently finished chapter for each series the user has read.
	query := `
		SELECT c.series_id, MAX(ucp.updated_at) as last_read_time
		FROM user_chapter_progress ucp
		JOIN chapters c ON ucp.chapter_id = c.id
		WHERE ucp.user_id = ?
		GROUP BY c.series_id
		ORDER BY last_read_time DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, userID, limit*2) // Fetch more to account for fully read series
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nextUpItems []*models.HomeSectionItem
	seriesIDs := make(map[int64]string)

	for rows.Next() {
		var seriesID int64
		var lastRead string
		if err := rows.Scan(&seriesID, &lastRead); err != nil {
			return nil, err
		}
		seriesIDs[seriesID] = lastRead
	}
	for seriesID, lastRead := range seriesIDs {
		var lastReadTime time.Time
		lastReadTime, err = time.Parse("2006-01-02 15:04:05", lastRead)
		if err != nil {
			log.Printf("GetNextUp: could not parse last read time %s: %v", lastRead, err)
			continue
		}

		// Step 2: For each series, find the next unread chapter.
		settings, err := s.GetSeriesSettings(seriesID, userID)
		var sortBy, sortDir string
		if err != nil {
			log.Printf("GetNextUp: could not get series settings for series %d: %v", seriesID, err)
			sortBy = "auto"
			sortDir = "asc"
		} else {
			sortBy = settings.SortBy
			sortDir = settings.SortDir
		}
		series, _, err := s.GetSeriesByID(seriesID, userID, 1, 9999, "", sortBy, sortDir) // Fetch all chapters
		if err != nil {
			log.Printf("GetNextUp: could not get series by ID %d: %v", seriesID, err)
			continue
		}

		// Find the first unread chapter
		var nextChapter *models.Chapter
		for _, ch := range series.Chapters {
			if ch.ProgressPercent > 0 && ch.ProgressPercent < 100 {
				break
			}
			if ch.ProgressPercent == 0 {
				nextChapter = ch
				break
			}
		}

		if nextChapter != nil {
			coverArt := series.CustomCoverURL
			if coverArt == "" {
				coverArt = series.Thumbnail
			}
			item := &models.HomeSectionItem{
				SeriesID:     series.ID,
				SeriesTitle:  series.Title,
				ChapterID:    &nextChapter.ID,
				ChapterTitle: nextChapter.Path,
				CoverArt:     coverArt,
				UpdatedAt:    lastReadTime, // Use this for final sorting
			}
			nextUpItems = append(nextUpItems, item)
		}
	}

	// Step 3: Sort the final list by the last read time and take the limit.
	sort.Slice(nextUpItems, func(i, j int) bool {
		return nextUpItems[i].UpdatedAt.After(nextUpItems[j].UpdatedAt)
	})

	if len(nextUpItems) > limit {
		return nextUpItems[:limit], nil
	}
	return nextUpItems, nil
}

// GetRecentlyAdded fetches recently added chapters and groups them by series.
func (s *Store) GetRecentlyAdded(limit int) ([]*models.HomeSectionItem, error) {
	query := `
		SELECT
			s.id, s.title, COALESCE(s.custom_cover_url, s.thumbnail, '') as cover_art,
			c.id as chapter_id, c.path as chapter_title, c.created_at
		FROM chapters c
		JOIN series s ON c.series_id = s.id
		ORDER BY c.created_at DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, limit*2) // Fetch more to ensure we have enough unique series
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seriesMap := make(map[int64]*models.HomeSectionItem)
	var orderedSeriesIDs []int64

	for rows.Next() {
		var item models.HomeSectionItem
		var chapterID int64
		if err := rows.Scan(&item.SeriesID, &item.SeriesTitle, &item.CoverArt, &chapterID, &item.ChapterTitle, &item.UpdatedAt); err != nil {
			return nil, err
		}

		if existing, ok := seriesMap[item.SeriesID]; ok {
			// This is the second (or more) chapter for this series.
			// Increment the count and clear the chapter-specific details.
			existing.NewChapterCount++
			existing.ChapterID = nil
			existing.ChapterTitle = ""
		} else {
			// This is the first time we've seen this series in the results.
			item.NewChapterCount = 1
			item.ChapterID = &chapterID // Keep the chapter details for now
			seriesMap[item.SeriesID] = &item
			orderedSeriesIDs = append(orderedSeriesIDs, item.SeriesID)
		}
	}

	var finalItems []*models.HomeSectionItem
	for _, seriesID := range orderedSeriesIDs {
		finalItems = append(finalItems, seriesMap[seriesID])
		if len(finalItems) >= limit {
			break
		}
	}

	return finalItems, nil
}

// GetStartReading fetches series the user has not started reading yet.
func (s *Store) GetStartReading(userID int64, limit int) ([]*models.HomeSectionItem, error) {
	query := `
		SELECT
			s.id, s.title, COALESCE(s.custom_cover_url, s.thumbnail, '') as cover_art, s.created_at
		FROM series s
		WHERE NOT EXISTS (
			SELECT 1 FROM user_chapter_progress ucp
			JOIN chapters c ON ucp.chapter_id = c.id
			WHERE c.series_id = s.id AND ucp.user_id = ?
		)
		ORDER BY s.created_at DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*models.HomeSectionItem
	for rows.Next() {
		var item models.HomeSectionItem
		if err := rows.Scan(&item.SeriesID, &item.SeriesTitle, &item.CoverArt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}
	return items, nil
}

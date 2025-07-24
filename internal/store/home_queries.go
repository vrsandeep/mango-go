package store

import (
	"database/sql"
	"sort"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/util"
)

// GetContinueReading fetches chapters the user has started but not finished, only one per series.
func (s *Store) GetContinueReading(userID int64, limit int) ([]*models.HomeSectionItem, error) {
	// --COALESCE(f.custom_cover_url, f.thumbnail, '') as cover_art,
	query := `
		SELECT *
		FROM (
			SELECT
				f.id as series_id,
				f.name as series_title,
				c.id as chapter_id,
				c.path as chapter_title,
				COALESCE(c.thumbnail, f.thumbnail, '') as cover_art,
				ucp.progress_percent,
				ucp.read,
				ucp.updated_at,
				ROW_NUMBER() OVER (PARTITION BY f.id ORDER BY ucp.updated_at DESC) as rn
			FROM user_chapter_progress ucp
			JOIN chapters c ON ucp.chapter_id = c.id
			JOIN folders f ON c.folder_id = f.id
			WHERE ucp.user_id = ? AND ucp.read = 0 AND ucp.progress_percent > 0
		)
		WHERE rn = 1
		ORDER BY updated_at DESC
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
		if err := rows.Scan(&item.SeriesID, &item.SeriesTitle, &chapterID, &item.ChapterTitle, &item.CoverArt, &progress, &read, &item.UpdatedAt, new(int)); err != nil {
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
	// Simple approach: Get all folders that the user has read chapters in
	query := `
		SELECT DISTINCT f.id, f.name, f.thumbnail, MAX(ucp.updated_at) as last_read_time
		FROM folders f
		JOIN chapters c ON c.folder_id = f.id
		JOIN user_chapter_progress ucp ON ucp.chapter_id = c.id
		WHERE ucp.user_id = ? AND ucp.read = 1
		GROUP BY f.id, f.name, f.thumbnail
		ORDER BY last_read_time DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, userID, limit*2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type folderInfo struct {
		id           int64
		name         string
		thumbnail    string
		lastReadTime time.Time
	}
	var folders []folderInfo

	for rows.Next() {
		var f folderInfo
		var lastRead string
		var thumbnail sql.NullString
		if err := rows.Scan(&f.id, &f.name, &thumbnail, &lastRead); err != nil {
			continue
		}
		f.lastReadTime, _ = time.Parse("2006-01-02 15:04:05", lastRead)
		f.thumbnail = thumbnail.String
		folders = append(folders, f)
	}

	var items []*models.HomeSectionItem
	for _, folder := range folders {
		// Find the first unread chapter in this folder
		nextChapter, err := s.findNextChapterInFolder(userID, folder.id)
		if err != nil {
			// If no unread chapters in this folder, try to find the next folder in the hierarchy
			nextChapter, err = s.findNextChapterInSiblingFolder(userID, folder.id)
			if err != nil {
				continue
			}
		}

		if nextChapter != nil {
			// Get the folder info for the chapter
			chapterFolder, err := s.GetFolder(nextChapter.FolderID)
			if err != nil {
				continue
			}
			items = append(items, &models.HomeSectionItem{
				SeriesID:     chapterFolder.ID,
				SeriesTitle:  chapterFolder.Name,
				ChapterID:    &nextChapter.ID,
				ChapterTitle: GetChapterTitle(nextChapter),
				CoverArt:     nextChapter.Thumbnail,
				UpdatedAt:    folder.lastReadTime,
			})
			if len(items) >= limit {
				break
			}
		}
	}

	return items, nil
}

// findNextChapterInFolder finds the next unread chapter in a specific folder
func (s *Store) findNextChapterInFolder(userID, folderID int64) (*models.Chapter, error) {
	query := `
		SELECT c.id, c.folder_id, c.path, c.thumbnail, COALESCE(ucp.read, 0) as read
		FROM chapters c
		LEFT JOIN user_chapter_progress ucp ON c.id = ucp.chapter_id AND ucp.user_id = ?
		WHERE c.folder_id = ?
		ORDER BY c.path ASC
	`
	rows, err := s.db.Query(query, userID, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var chapter models.Chapter
		var read bool
		if err := rows.Scan(&chapter.ID, &chapter.FolderID, &chapter.Path, &chapter.Thumbnail, &read); err != nil {
			continue
		}
		if !read {
			return &chapter, nil
		}
	}

	return nil, sql.ErrNoRows
}

// findNextChapterInSiblingFolder finds the next unread chapter in a sibling folder
func (s *Store) findNextChapterInSiblingFolder(userID, folderID int64) (*models.Chapter, error) {
	// Get the parent folder of the current folder
	var parentID *int64
	err := s.db.QueryRow("SELECT parent_id FROM folders WHERE id = ?", folderID).Scan(&parentID)
	if err != nil {
		return nil, err
	}

	if parentID == nil {
		// This is a root folder, no siblings to check
		return nil, sql.ErrNoRows
	}

	// Get all sibling folders (folders with the same parent)
	query := `
		SELECT id, name FROM folders
		WHERE parent_id = ?
		ORDER BY name ASC
	`
	rows, err := s.db.Query(query, *parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var siblingFolders []struct {
		id   int64
		name string
	}

	for rows.Next() {
		var sibling struct {
			id   int64
			name string
		}
		if err := rows.Scan(&sibling.id, &sibling.name); err != nil {
			continue
		}
		siblingFolders = append(siblingFolders, sibling)
	}

	sort.Slice(siblingFolders, func(i, j int) bool {
		return util.NaturalSortLess(siblingFolders[i].name, siblingFolders[j].name)
	})

	// Find the current folder in the list and get the next one
	for i, sibling := range siblingFolders {
		if sibling.id == folderID {
			// Check the next sibling folder
			if i+1 < len(siblingFolders) {
				nextSibling := siblingFolders[i+1]
				// Find the first unread chapter in the next sibling folder
				return s.findNextChapterInFolder(userID, nextSibling.id)
			}
			break
		}
	}

	// If no next sibling found, try the parent's siblings
	return s.findNextChapterInSiblingFolder(userID, *parentID)
}

// GetRecentlyAdded fetches recently added chapters and groups them by series.
func (s *Store) GetRecentlyAdded(limit int) ([]*models.HomeSectionItem, error) {
	query := `
		SELECT
			f.id as series_id,
			f.name as series_title,
			f.thumbnail as cover_art,
			c.id as chapter_id,
			c.path as chapter_title,
			c.created_at
		FROM chapters c
		JOIN folders f ON f.id = c.folder_id
		ORDER BY c.created_at DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, limit*2) // Fetch more to ensure we have enough unique series
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	folderMap := make(map[int64]*models.HomeSectionItem)
	var orderedFolderIDs []int64

	for rows.Next() {
		var item models.HomeSectionItem
		var chapterID int64
		var thumbnail sql.NullString
		var chapterTitle string
		if err := rows.Scan(&item.SeriesID, &item.SeriesTitle, &thumbnail, &chapterID, &chapterTitle, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.CoverArt = thumbnail.String

		if existing, ok := folderMap[item.SeriesID]; ok {
			// This is the second (or more) chapter for this series.
			// Increment the count and clear the chapter-specific details.
			existing.NewChapterCount++
			existing.ChapterID = nil
			existing.ChapterTitle = ""
		} else {
			// This is the first time we've seen this series in the results.
			item.NewChapterCount = 1
			item.ChapterID = &chapterID // Keep the chapter details for now
			item.ChapterTitle = GetChapterTitle(&models.Chapter{Path: chapterTitle})
			folderMap[item.SeriesID] = &item
			orderedFolderIDs = append(orderedFolderIDs, item.SeriesID)
		}
	}

	var finalItems []*models.HomeSectionItem
	for _, seriesID := range orderedFolderIDs {
		finalItems = append(finalItems, folderMap[seriesID])
		if len(finalItems) >= limit {
			break
		}
	}

	return finalItems, nil
}

// GetStartReading fetches top-level folders which the user has not started reading yet.
func (s *Store) GetStartReading(userID int64, limit int) ([]*models.HomeSectionItem, error) {
	query := `
		SELECT
			f.id,
			f.name,
			COALESCE(f.thumbnail, '') as cover_art,
			f.created_at
		FROM folders f
		WHERE
			f.parent_id IS NULL AND NOT EXISTS (
				SELECT 1 FROM (
					WITH RECURSIVE folder_subtree(id) AS (
						SELECT f.id
						UNION ALL
						SELECT sub.id FROM folders sub JOIN folder_subtree st ON sub.parent_id = st.id
					)
					SELECT id FROM folder_subtree
				) subtree
				JOIN chapters c ON c.folder_id = subtree.id
				JOIN user_chapter_progress ucp ON ucp.chapter_id = c.id
				WHERE ucp.user_id = ?
			)
		ORDER BY f.created_at DESC
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

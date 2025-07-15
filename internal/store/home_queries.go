package store

import (
	"database/sql"
	"log"
	"sort"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

// GetContinueReading fetches chapters the user has started but not finished, only one per series.
func (s *Store) GetContinueReading(userID int64, limit int) ([]*models.HomeSectionItem, error) {
	// --COALESCE(f.custom_cover_url, f.thumbnail, '') as cover_art,
	query := `
		SELECT *
		FROM (
			SELECT
				f.id as series_id,
				f.title as series_title,
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
	// Step 1: Find the most recently read chapter for each top-level folder tree.
	// This gives us our starting points (lastReadChapterID) and the timestamp for final sorting.
	query := `
		WITH RECURSIVE folder_root AS (
			SELECT id, id as root_id, parent_id FROM folders WHERE parent_id IS NULL
			UNION ALL
			SELECT f.id, fr.root_id, f.parent_id FROM folders f JOIN folder_root fr ON f.parent_id = fr.id
		)
		SELECT
			MAX(ucp.updated_at) as last_read_time,
			(SELECT chapter_id FROM user_chapter_progress ucp2
				WHERE ucp2.user_id = ucp.user_id
				AND ucp2.updated_at = MAX(ucp.updated_at)
			) as last_read_chapter_id
		FROM user_chapter_progress ucp
		JOIN chapters c ON ucp.chapter_id = c.id
		JOIN folder_root fr ON c.folder_id = fr.id
		WHERE ucp.user_id = ? AND ucp.read = 1
		GROUP BY fr.root_id
		ORDER BY last_read_time DESC
		LIMIT ?
	`
	rows, err := s.db.Query(query, userID, limit*2) // Fetch more to account for fully read series
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type suggestionCandidate struct {
		item     *models.HomeSectionItem
		lastRead time.Time
	}
	var candidates []suggestionCandidate

	folderIds := make(map[int64]string)
	for rows.Next() {
		var lastReadChapterID int64
		var lastRead string
		if err := rows.Scan(&lastReadChapterID, &lastRead); err != nil {
			log.Printf("GetNextUp: could not scan row: %v", err)
			continue
		}
		folderIds[lastReadChapterID] = lastRead
	}

	for lastReadChapterID, lastRead := range folderIds {

		var lastReadTime time.Time
		lastReadTime, err = time.Parse("2006-01-02 15:04:05", lastRead)
		if err != nil {
			log.Printf("GetNextUp: could not parse last read time %s: %v", lastRead, err)
			continue
		}

		// Step 2: For each chapter, find the next unread chapter.
		nextUpItem, err := s.findNextUpItem(userID, lastReadChapterID)
		if err == nil && nextUpItem != nil {
			candidates = append(candidates, suggestionCandidate{item: nextUpItem, lastRead: lastReadTime})
		}

	}

	// Step 3: Sort the final list by the last read time and take the limit.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].lastRead.After(candidates[j].lastRead)
	})

	var finalItems []*models.HomeSectionItem
	for i, c := range candidates {
		if i >= limit {
			break
		}
		finalItems = append(finalItems, c.item)
	}

	return finalItems, nil
}

// findNextUpItem is a helper function that contains the core traversal logic.
func (s *Store) findNextUpItem(userID, lastReadChapterID int64) (*models.HomeSectionItem, error) {
	// Get details of the last-read chapter
	lastReadChapter, err := s.GetChapterByID(lastReadChapterID, userID)
	if err != nil {
		return nil, err
	}

	// 1. Look for the next chapter in the same folder.
	opts := ListItemsOptions{
		UserID:   userID,
		ParentID: &lastReadChapter.FolderID,
		SortBy:   "auto",
		SortDir:  "asc",
		Page:     1,
		PerPage:  9999, // Get all chapters in this folder
	}
	_, _, chapters, _, _ := s.ListItems(opts)
	for i, ch := range chapters {
		if ch.ID == lastReadChapter.ID {
			if i+1 < len(chapters) {
				nextCh := chapters[i+1]
				if !nextCh.Read {
					folder, _ := s.GetFolder(nextCh.FolderID)
					return &models.HomeSectionItem{
						SeriesID:        folder.ID,
						SeriesTitle:     folder.Name,
						ChapterID:       &nextCh.ID,
						ChapterTitle:    nextCh.Path,
						CoverArt:        nextCh.Thumbnail,
						ProgressPercent: &nextCh.ProgressPercent,
						Read:            &nextCh.Read,
					}, nil
				}
			}
			break // Found our spot, no need to look further in this folder.
		}
	}

	// 2. If no next chapter in the same folder, traverse up the tree.
	// This is a recursive search up the folder hierarchy.
	currentFolder, err := s.GetFolder(lastReadChapter.FolderID)
	if err != nil {
		return nil, err
	}
	// Start the recursive search from the parent of the folder we just finished.
	return s.findNextItemInParent(userID, currentFolder.ParentID, currentFolder.ID)
}

// findNextItemInParent looks for the next sibling of a given child (folder/chapter)
// and finds the first chapter within it.
func (s *Store) findNextItemInParent(userID int64, parentID *int64, previousChildID int64) (*models.HomeSectionItem, error) {
	if parentID == nil {
		// We've reached the root of the library, no more siblings to check.
		return nil, sql.ErrNoRows
	}

	// Get all items (subfolders and chapters) in the parent, sorted naturally.
	_, subfolders, _, _, _ := s.ListItems(ListItemsOptions{UserID: userID, ParentID: parentID, PerPage: 9999, SortBy: "auto", SortDir: "asc"})

	// Combine and find the index of the folder we just came from.
	type item struct {
		id       int64
		isFolder bool
	}
	var allItems []item
	for _, f := range subfolders {
		allItems = append(allItems, item{id: f.ID, isFolder: true})
	}
	// We don't need to check for chapters at this level, as we're looking for the next *container*.

	for i, itm := range allItems {
		if itm.id == previousChildID {
			// Found the folder we finished. Check the next item in the list.
			if i+1 < len(allItems) {
				nextItem := allItems[i+1]
				if nextItem.isFolder {
					// The next item is a folder, find the first chapter inside its entire subtree.
					return s.findFirstChapterInSubtree(userID, nextItem.id)
				}
			}
			break
		}
	}

	// If we didn't find a next sibling, recurse up to the parent's parent.
	parentFolder, _ := s.GetFolder(*parentID)
	return s.findNextItemInParent(userID, parentFolder.ParentID, parentFolder.ID)
}

// findFirstChapterInSubtree performs a recursive, depth-first search for the
// first unread chapter within a given folder ID.
func (s *Store) findFirstChapterInSubtree(userID int64, folderID int64) (*models.HomeSectionItem, error) {
	_, subfolders, chapters, _, _ := s.ListItems(ListItemsOptions{UserID: userID, ParentID: &folderID, PerPage: 9999, SortBy: "auto", SortDir: "asc"})

	// First, check for chapters directly in this folder.
	for _, ch := range chapters {
		if !ch.Read {
			folder, _ := s.GetFolder(ch.FolderID)
			return &models.HomeSectionItem{
				SeriesID:     folder.ID,
				SeriesTitle:  folder.Name,
				ChapterID:    &ch.ID,
				ChapterTitle: ch.Path,
				CoverArt:     ch.Thumbnail,
			}, nil
		}
	}

	// If no chapters here, recursively check the first subfolder.
	for _, sf := range subfolders {
		item, err := s.findFirstChapterInSubtree(userID, sf.ID)
		if err == nil && item != nil {
			return item, nil // Found it in a subfolder.
		}
	}

	return nil, sql.ErrNoRows
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
		if err := rows.Scan(&item.SeriesID, &item.SeriesTitle, &thumbnail, &chapterID, &item.ChapterTitle, &item.UpdatedAt); err != nil {
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

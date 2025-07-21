// A NEW file in the store package dedicated to read queries for the API.
// This keeps the original store.go focused on write/update operations.

package store

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/util"
)

func (s *Store) getChaptersForFolder(folderId int64, userID int64, page, perPage int, search, sortBy, sortDir string) ([]*models.Chapter, error) {
	chapterQuery := `
		SELECT c.id, c.path, c.page_count,
		       COALESCE(ucp.read, 0) as read,
		       COALESCE(ucp.progress_percent, 0) as progress_percent,
		       c.thumbnail,
			   c.created_at,
			   c.updated_at
		FROM chapters c
		LEFT JOIN user_chapter_progress ucp ON c.id = ucp.chapter_id AND ucp.user_id = ?
		WHERE c.folder_id = ?
	`
	var chapterArgs []interface{}
	chapterArgs = append(chapterArgs, userID, folderId)
	if search != "" {
		chapterQuery += " AND c.path LIKE ?"
		chapterArgs = append(chapterArgs, "%"+search+"%")
	}
	sortDir = strings.ToUpper(sortDir)
	if sortDir != "ASC" && sortDir != "DESC" {
		sortDir = "ASC"
	}
	switch sortBy {
	case "path":
		chapterQuery += fmt.Sprintf(" ORDER BY c.path %s LIMIT ? OFFSET ?", sortDir)
	case "auto":
		// Auto sort is handled in Go after fetching
		chapterQuery += " ORDER BY c.path ASC"
	default:
		chapterQuery += " ORDER BY c.path ASC LIMIT ? OFFSET ?"
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
		if err := rows.Scan(&chapter.ID, &chapter.Path, &chapter.PageCount, &chapter.Read, &chapter.ProgressPercent, &chapThumb, &chapter.CreatedAt, &chapter.UpdatedAt); err != nil {
			return nil, err
		}
		chapter.Thumbnail = chapThumb.String
		chapter.FolderID = folderId
		chapters = append(chapters, &chapter)
	}
	if sortBy == "auto" {
		// Use the ChapterSorter to sort chapters
		chapterTitles := make([]string, len(chapters))
		for i, chapter := range chapters {
			chapterTitles[i] = GetChapterTitle(chapter)
		}
		cs := util.NewChapterSorter(chapterTitles)
		slices.SortFunc(chapters, func(a, b *models.Chapter) int {
			comparison := cs.Compare(GetChapterTitle(a), GetChapterTitle(b))
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

// GetChapterTitle extracts the title from a chapter's path.
func GetChapterTitle(chapter *models.Chapter) string {
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

// GetChapterNeighbors finds the previous and next chapter IDs based on sort settings.
func (s *Store) GetChapterNeighbors(folderID, currentChapterID, userID int64) (map[string]*int64, error) {
	var chapters []*models.Chapter
	chapters, err := s.getChaptersForFolder(folderID, userID, 1, 10000, "", "auto", "asc")
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

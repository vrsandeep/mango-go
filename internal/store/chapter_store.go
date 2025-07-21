package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

var ErrChapterNotFound = errors.New("chapter not found")

type ChapterInfo struct {
	ID   int64
	Path string
}

// CreateChapter inserts a new chapter record into the database.
func (s *Store) CreateChapter(folderID int64, path, hash string, pageCount int, thumbnail string) (*models.Chapter, error) {
	query := "INSERT INTO chapters (folder_id, path, content_hash, page_count, thumbnail, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)"
	now := time.Now()
	res, err := s.db.Exec(query, folderID, path, hash, pageCount, thumbnail, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.Chapter{ID: id, FolderID: folderID, Path: path, ContentHash: hash, PageCount: pageCount}, nil
}

// GetChapterByID fetches a single chapter by its ID.
func (s *Store) GetChapterByID(id int64, userID int64) (*models.Chapter, error) {
	var chapter models.Chapter
	var thumb sql.NullString
	query := `
		SELECT c.id, c.folder_id, c.path, c.content_hash, c.page_count,
		       COALESCE(ucp.read, 0) as read,
		       COALESCE(ucp.progress_percent, 0) as progress_percent,
		       c.thumbnail,
			   c.created_at,
			   c.updated_at
		FROM chapters c
		LEFT JOIN user_chapter_progress ucp ON c.id = ucp.chapter_id AND ucp.user_id = ?
		WHERE c.id = ?
	`
	err := s.db.QueryRow(query, userID, id).Scan(
		&chapter.ID, &chapter.FolderID, &chapter.Path, &chapter.ContentHash, &chapter.PageCount,
		&chapter.Read, &chapter.ProgressPercent,
		&thumb, &chapter.CreatedAt, &chapter.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	chapter.Thumbnail = thumb.String
	return &chapter, nil
}

// GetAllChaptersByHash retrieves all chapters and maps them by their content hash for efficient lookup.
func (s *Store) GetAllChaptersByHash() (map[string]ChapterInfo, error) {
	rows, err := s.db.Query("SELECT id, path, content_hash FROM chapters")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chapterMap := make(map[string]ChapterInfo)
	for rows.Next() {
		var info ChapterInfo
		var hash sql.NullString
		if err := rows.Scan(&info.ID, &info.Path, &hash); err != nil {
			return nil, err
		}
		if hash.Valid {
			chapterMap[hash.String] = info
		}
	}
	return chapterMap, nil
}

// UpdateChapterPath updates a chapter's path and folder ID when it has been moved.
func (s *Store) UpdateChapterPath(id int64, newPath string, newFolderID int64) error {
	query := "UPDATE chapters SET path = ?, folder_id = ?, updated_at = ? WHERE id = ?"
	_, err := s.db.Exec(query, newPath, newFolderID, time.Now(), id)
	return err
}

// DeleteChapterByHash removes a chapter from the database using its unique content hash.
func (s *Store) DeleteChapterByHash(hash string) error {
	_, err := s.db.Exec("DELETE FROM chapters WHERE content_hash = ?", hash)
	return err
}

// UpdateChapterThumbnail updates the thumbnail for a single chapter.
func (s *Store) UpdateChapterThumbnail(chapterID int64, thumbnail string) error {
	result, err := s.db.Exec("UPDATE chapters SET thumbnail = ? WHERE id = ?", thumbnail, chapterID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrChapterNotFound
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
	result, err := s.db.Exec(query, userID, chapterID, progressPercent, read)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrChapterNotFound
	}
	return nil
}

func (s *Store) GetFolderStats(folderID int64, userID int64) (int, int, error) {
	query := `
		WITH total_chapters AS (
			SELECT COUNT(*) as total_chapters, ? as folder_id
			FROM chapters c
			WHERE c.folder_id = ?
		),
		read_chapters AS (
			SELECT COUNT(*) as read_chapters, ? as folder_id
			FROM chapters c
			LEFT JOIN user_chapter_progress ucp ON c.id = ucp.chapter_id
			WHERE c.folder_id = ? AND ucp.user_id = ?
		)
		SELECT COALESCE(total_chapters.total_chapters, 0) as total_chapters, COALESCE(read_chapters.read_chapters, 0) as read_chapters
		FROM total_chapters
		LEFT JOIN read_chapters ON total_chapters.folder_id = read_chapters.folder_id
	`
	var totalChapters int
	var readChapters int
	err := s.db.QueryRow(query, folderID, folderID, folderID, folderID, userID).Scan(&totalChapters, &readChapters)
	if err != nil {
		return 0, 0, err
	}
	return totalChapters, readChapters, nil
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
	// 1. Fetch all chapters in the folder
	_, _, chapters, _, err := s.ListItems(ListItemsOptions{
		UserID:   userID,
		ParentID: &folderID,
		Page:     1,
		PerPage:  10000, // Assume a folder won't have more chapters than this
		SortBy:   "auto",
		SortDir:  "asc",
	})
	if err != nil {
		return nil, err
	}

	// 2. Find the index of the current chapter
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

package store

import (
	"database/sql"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

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

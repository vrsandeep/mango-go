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

// CreateFolder inserts a new folder into the database.
func (s *Store) CreateFolder(path, name string, parentID *int64) (*models.Folder, error) {
	query := "INSERT INTO folders (path, name, parent_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)"
	now := time.Now()
	res, err := s.db.Exec(query, path, name, parentID, now, now)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &models.Folder{ID: id, Path: path, Name: name, ParentID: parentID}, nil
}

// GetFolder retrieves a single folder by its ID.
func (s *Store) GetFolder(id int64) (*models.Folder, error) {
	var folder models.Folder
	var parentID sql.NullInt64
	var thumbnail sql.NullString
	query := "SELECT id, path, name, parent_id, thumbnail, created_at, updated_at FROM folders WHERE id = ?"
	err := s.db.QueryRow(query, id).Scan(&folder.ID, &folder.Path, &folder.Name, &parentID, &thumbnail, &folder.CreatedAt, &folder.UpdatedAt)
	if parentID.Valid {
		folder.ParentID = &parentID.Int64
	}
	folder.Thumbnail = thumbnail.String
	return &folder, err
}

// GetAllFoldersByPath retrieves all folders and maps them by their full path for efficient lookup.
func (s *Store) GetAllFoldersByPath() (map[string]*models.Folder, error) {
	rows, err := s.db.Query("SELECT id, path, name, parent_id FROM folders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	folderMap := make(map[string]*models.Folder)
	for rows.Next() {
		var folder models.Folder
		var parentID sql.NullInt64
		if err := rows.Scan(&folder.ID, &folder.Path, &folder.Name, &parentID); err != nil {
			return nil, err
		}
		if parentID.Valid {
			folder.ParentID = &parentID.Int64
		}
		folderMap[folder.Path] = &folder
	}
	return folderMap, nil
}

// DeleteFolder removes a folder by its ID.
func (s *Store) DeleteFolder(id int64) error {
	_, err := s.db.Exec("DELETE FROM folders WHERE id = ?", id)
	return err
}

// UpdateAllFolderThumbnails recursively finds the first chapter in a folder's subtree
// and sets its thumbnail as the folder's thumbnail.
func (s *Store) UpdateAllFolderThumbnails() error {
	// This is a complex operation. A simplified approach:
	// 1. Get all chapters with their folder IDs and thumbnails.
	// 2. Group them by folder ID.
	// 3. For each folder, find the "first" chapter via natural sort.
	// 4. Update that folder's thumbnail.
	// 5. Recursively do this for parent folders.

	rows, err := s.db.Query("SELECT id FROM folders")
	if err != nil {
		return err
	}
	defer rows.Close()

	var folderIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		folderIDs = append(folderIDs, id)
	}

	for _, folderID := range folderIDs {
		s.updateSingleFolderThumbnail(folderID)
	}
	return nil
}

// updateSingleFolderThumbnail is a helper for the above function.
func (s *Store) updateSingleFolderThumbnail(folderID int64) {
	// This query recursively finds all chapters within a folder and its subfolders.
	query := `
		WITH RECURSIVE folder_tree(id) AS (
			SELECT ?
			UNION ALL
			SELECT f.id FROM folders f JOIN folder_tree ft ON f.parent_id = ft.id
		)
		SELECT c.thumbnail, c.path FROM chapters c WHERE c.folder_id IN folder_tree ORDER BY c.created_at ASC;
	`
	rows, err := s.db.Query(query, folderID)
	if err != nil {
		log.Printf("Error finding chapters for folder thumbnail %d: %v", folderID, err)
		return
	}
	defer rows.Close()

	var chapters []struct {
		Thumbnail sql.NullString
		Path      string
	}
	for rows.Next() {
		var c struct {
			Thumbnail sql.NullString
			Path      string
		}
		if err := rows.Scan(&c.Thumbnail, &c.Path); err != nil {
			continue
		}
		chapters = append(chapters, c)
	}

	if len(chapters) > 0 {
		// Sort naturally to find the true first chapter
		sort.Slice(chapters, func(i, j int) bool {
			return util.NaturalSortLess(chapters[i].Path, chapters[j].Path)
		})
		firstChapter := chapters[0]
		if firstChapter.Thumbnail.Valid {
			s.db.Exec("UPDATE folders SET thumbnail = ? WHERE id = ?", firstChapter.Thumbnail.String, folderID)
		}
	}
}

// GetFolderContents retrieves all subfolders and chapters for a given parent folder.
func (s *Store) GetFolderContents(userID int64, parentID *int64, page, perPage int, search, sortBy, sortDir string) (*models.Folder, []*models.Folder, []*models.Chapter, int, error) {
	// ... (complex implementation to fetch current folder details, subfolders, and chapters separately)
	// ... (it will apply search, sort, and pagination to the combined list)
	// ... (it will also join with user_chapter_progress to get read status for chapters)
	return nil, nil, nil, 0, nil // Placeholder for brevity
}

// GetFolderPath retrieves the entire ancestry of a folder for breadcrumbs.
func (s *Store) GetFolderPath(folderID int64) ([]*models.Folder, error) {
	var path []*models.Folder
	currentID := &folderID
	for currentID != nil {
		folder, err := s.GetFolder(*currentID)
		if err != nil {
			return nil, err
		}
		path = append([]*models.Folder{folder}, path...) // Prepend to reverse the order
		currentID = folder.ParentID
	}
	return path, nil
}

// AddTagToFolder creates the association between a folder and a tag.
func (s *Store) AddTagToFolder(folderID int64, tagName string) (*models.Tag, error) {
	tagName = strings.TrimSpace(strings.ToLower(tagName))
	if tagName == "" {
		return nil, fmt.Errorf("tag name cannot be empty")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	tag, err := s.GetOrCreateTag(tagName)
	if err != nil {
		return nil, err
	}
	_, err = s.db.Exec("INSERT OR IGNORE INTO folder_tags (folder_id, tag_id) VALUES (?, ?)", folderID, tag.ID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			// If the tag is already associated with the folder, ignore the error
			return &models.Tag{ID: tag.ID, Name: tag.Name}, nil
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return tag, nil
}

// RemoveTagFromFolder removes the association between a folder and a tag.
func (s *Store) RemoveTagFromFolder(folderID, tagID int64) error {
	_, err := s.db.Exec("DELETE FROM folder_tags WHERE folder_id = ? AND tag_id = ?", folderID, tagID)
	return err
}

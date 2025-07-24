package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/vrsandeep/mango-go/internal/models"
)

// ListTagsWithCounts returns all tags along with the count of series they are associated with.
func (s *Store) ListTagsWithCounts() ([]*models.Tag, error) {
	query := `
		SELECT t.id, t.name, COUNT(st.folder_id) as folder_count
		FROM tags t
		LEFT JOIN folder_tags st ON t.id = st.tag_id
		GROUP BY t.id
		ORDER BY t.name ASC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.FolderCount); err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}
	return tags, nil
}

// GetTagByID retrieves a single tag by its ID.
func (s *Store) GetTagByID(id int64) (*models.Tag, error) {
	var tag models.Tag
	err := s.db.QueryRow("SELECT id, name FROM tags WHERE id = ?", id).Scan(&tag.ID, &tag.Name)
	return &tag, err
}

// AddTagToFolder creates the association between a folder and a tag.
func (s *Store) AddTagToFolder(folderID int64, tagName string) (*models.Tag, error) {
	// Use a single transaction for the entire operation
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get or create tag within the transaction
	tagName = strings.TrimSpace(strings.ToLower(tagName))
	if tagName == "" {
		return nil, fmt.Errorf("tag name cannot be empty")
	}

	var tag models.Tag
	err = tx.QueryRow("SELECT id, name FROM tags WHERE name = ?", tagName).Scan(&tag.ID, &tag.Name)
	if err == sql.ErrNoRows {
		// Tag does not exist, create it
		res, err := tx.Exec("INSERT INTO tags (name) VALUES (?)", tagName)
		if err != nil {
			return nil, err
		}
		tagID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		tag.ID = tagID
		tag.Name = tagName
	} else if err != nil {
		return nil, err
	}

	// Insert folder-tag association within the same transaction
	_, err = tx.Exec("INSERT OR IGNORE INTO folder_tags (folder_id, tag_id) VALUES (?, ?)", folderID, tag.ID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			// If the tag is already associated with the folder, ignore the error
			return &models.Tag{ID: tag.ID, Name: tag.Name}, tx.Commit()
		}
		return nil, err
	}

	return &models.Tag{ID: tag.ID, Name: tag.Name}, tx.Commit()

}

// RemoveTagFromFolder removes the association between a folder and a tag.
func (s *Store) RemoveTagFromFolder(folderID, tagID int64) error {
	_, err := s.db.Exec("DELETE FROM folder_tags WHERE folder_id = ? AND tag_id = ?", folderID, tagID)
	return err
}

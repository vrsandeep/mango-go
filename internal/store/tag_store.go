package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/vrsandeep/mango-go/internal/models"
)

func (s *Store) GetOrCreateTag(name string) (*models.Tag, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return nil, fmt.Errorf("tag name cannot be empty")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var tag models.Tag
	err = s.db.QueryRow("SELECT id, name FROM tags WHERE name = ?", name).Scan(&tag.ID, &tag.Name)
	if err == sql.ErrNoRows {
		// Tag does not exist, create it
		res, err := s.db.Exec("INSERT INTO tags (name) VALUES (?)", name)
		if err != nil {
			return nil, err
		}
		tagID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		tag.ID = tagID
		tag.Name = name
	} else if err != nil {
		return nil, err
	}
	return &tag, tx.Commit()
}

// AddTagToFolder creates the association between a folder and a tag.
func (s *Store) AddTagToFolder(folderID int64, tagName string) (*models.Tag, error) {
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
	return tag, nil
}

// RemoveTagFromFolder removes the association between a folder and a tag.
func (s *Store) RemoveTagFromFolder(folderID, tagID int64) error {
	_, err := s.db.Exec("DELETE FROM folder_tags WHERE folder_id = ? AND tag_id = ?", folderID, tagID)
	return err
}

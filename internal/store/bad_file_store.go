// This file handles database operations for bad/corrupted archive files.

package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/vrsandeep/mango-go/internal/models"
)

// BadFileStore handles database operations for bad files.
type BadFileStore struct {
	db *sql.DB
}

// NewBadFileStore creates a new BadFileStore instance.
func NewBadFileStore(db *sql.DB) *BadFileStore {
	return &BadFileStore{db: db}
}

// CreateBadFile adds a new bad file entry to the database.
func (s *BadFileStore) CreateBadFile(path, error string, fileSize int64) error {
	fileName := filepath.Base(path)

	query := `
		INSERT OR REPLACE INTO bad_files (path, file_name, error, file_size, detected_at, last_checked)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query, path, fileName, error, fileSize, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to create bad file entry: %w", err)
	}

	return nil
}

// GetAllBadFiles retrieves all bad files from the database.
func (s *BadFileStore) GetAllBadFiles() ([]*models.BadFile, error) {
	query := `
		SELECT id, path, file_name, error, file_size, detected_at, last_checked
		FROM bad_files
		ORDER BY detected_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query bad files: %w", err)
	}
	defer rows.Close()

	// Initialize with an empty slice to ensure it's never nil
	badFiles := make([]*models.BadFile, 0)
	for rows.Next() {
		bf := &models.BadFile{}
		err := rows.Scan(&bf.ID, &bf.Path, &bf.FileName, &bf.Error, &bf.FileSize, &bf.DetectedAt, &bf.LastChecked)
		if err != nil {
			return nil, fmt.Errorf("failed to scan bad file row: %w", err)
		}
		badFiles = append(badFiles, bf)
	}

	return badFiles, nil
}

// DeleteBadFile removes a bad file entry by ID.
func (s *BadFileStore) DeleteBadFile(id int64) error {
	query := `DELETE FROM bad_files WHERE id = ?`

	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete bad file: %w", err)
	}

	return nil
}

// DeleteBadFileByPath removes a bad file entry by path.
func (s *BadFileStore) DeleteBadFileByPath(path string) error {
	query := `DELETE FROM bad_files WHERE path = ?`

	_, err := s.db.Exec(query, path)
	if err != nil {
		return fmt.Errorf("failed to delete bad file by path: %w", err)
	}

	return nil
}

// CountBadFiles returns the total number of bad files.
func (s *BadFileStore) CountBadFiles() (int, error) {
	query := `SELECT COUNT(*) FROM bad_files`

	var count int
	err := s.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count bad files: %w", err)
	}

	return count, nil
}

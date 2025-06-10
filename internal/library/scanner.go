// This file contains the main logic for scanning the library directory.
// It walks the directory tree, identifies manga archives, and uses other
// helper modules to parse them and extract metadata.

package library

import (
	"database/sql"
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/store"
)

// Scanner is responsible for scanning the library and updating the database.
type Scanner struct {
	cfg *config.Config
	db  *sql.DB
	st  *store.Store // The data access layer
}

// NewScanner creates a new Scanner instance.
func NewScanner(cfg *config.Config, db *sql.DB) *Scanner {
	return &Scanner{
		cfg: cfg,
		db:  db,
		st:  store.New(db),
	}
}

// Scan walks the configured library path and updates the database.
func (s *Scanner) Scan() error {
	return filepath.WalkDir(s.cfg.Library.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil // Skip directories
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".cbz" || ext == ".cbr" {
			log.Printf("Processing archive: %s", path)

			seriesTitle, _ := ExtractMetadataFromPath(path, s.cfg.Library.Path)

			// Begin a transaction for this file.
			tx, err := s.db.Begin()
			if err != nil {
				return err
			}
			// Defer a rollback in case of error. Commit will be called on success.
			defer tx.Rollback()

			// Get or create the manga series ID.
			seriesID, err := s.st.GetOrCreateSeries(tx, seriesTitle, filepath.Dir(path))
			if err != nil {
				return err
			}

			// Parse the archive to get page information.
			pages, err := ParseArchive(path)
			if err != nil {
				log.Printf("Warning: could not parse archive %s: %v", path, err)
				return nil // Continue scanning even if one file is corrupt
			}

			// Add or update the chapter in the database.
			_, err = s.st.AddOrUpdateChapter(tx, seriesID, path, len(pages))
			if err != nil {
				return err
			}

			// Commit the transaction if everything was successful.
			return tx.Commit()
		}
		return nil
	})
}

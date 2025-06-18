// This file contains the main logic for scanning the library directory.
// It walks the directory tree, identifies manga archives, and uses other
// helper modules to parse them and extract metadata.

package library

import (
	"database/sql"
	"log"
	"os"
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

// Scan accepts a set of paths to ignore (for incremental scans)
// and a channel to report progress.
func (s *Scanner) Scan(ignorePaths map[string]bool, progressChan chan<- float64) error {
	if progressChan != nil {
		defer close(progressChan)
	}
	var filesToScan []string
	err := filepath.WalkDir(s.cfg.Library.Path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(d.Name(), ".cbz") || strings.HasSuffix(d.Name(), ".cbr")) {
			filesToScan = append(filesToScan, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	totalFiles := len(filesToScan)
	for i, path := range filesToScan {
		if ignorePaths != nil {
			if _, exists := ignorePaths[path]; exists {
				continue // Skip already processed files in incremental scan
			}
		}
		err := s.processFile(path)
		if err != nil {
			return err
		}
		if progressChan != nil {
			progressChan <- (float64(i+1) / float64(totalFiles)) * 100
		}
	}
	err = s.st.UpdateAllSeriesThumbnails()
	if err != nil {
		return err
	}
	return nil

}

// processFile processes a single archive file, extracting metadata and updating the database.
func (s *Scanner) processFile(path string) error {
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
	pages, firstPageData, err := ParseArchive(path)
	if err != nil {
		log.Printf("Warning: could not parse archive %s: %v", path, err)
		return nil // Continue scanning even if one file is corrupt
	}

	var thumbnailData string
	if firstPageData != nil {
		thumbnailData, err = GenerateThumbnail(firstPageData)
		if err != nil {
			log.Printf("Warning: could not generate thumbnail for %s: %v", path, err)
		}
	}

	// Add or update the chapter in the database.
	_, err = s.st.AddOrUpdateChapter(tx, seriesID, path, len(pages), thumbnailData)
	if err != nil {
		return err
	}

	// If a thumbnail was generated, try to set it as the series cover.
	if thumbnailData != "" {
		if err := s.st.UpdateSeriesThumbnailIfNeeded(tx, seriesID, thumbnailData); err != nil {
			return err
		}
	}

	// Commit the transaction if everything was successful.
	return tx.Commit()
}

// This file contains the main logic for scanning the library directory.
// It walks the directory tree, identifies manga archives, and uses other
// helper modules to parse them and extract metadata.

package library

import (
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/models"
)

// ScanLibrary walks the configured library path and builds a collection of manga.
func ScanLibrary(cfg *config.Config) (map[string]*models.Manga, error) {
	mangaCollection := make(map[string]*models.Manga)

	err := filepath.WalkDir(cfg.LibraryPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil // Skip directories
		}

		// Check for supported archive file extensions
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".cbz" || ext == ".cbr" {
			log.Printf("Found potential archive: %s", path)

			// Extract metadata from the file path
			seriesTitle, chapterName := ExtractMetadataFromPath(path, cfg.LibraryPath)

			// Get or create the manga series in our collection
			manga, exists := mangaCollection[seriesTitle]
			if !exists {
				manga = &models.Manga{
					Title:    seriesTitle,
					Path:     filepath.Dir(path),
					Chapters: []*models.Chapter{},
				}
				mangaCollection[seriesTitle] = manga
			}

			// Parse the archive to get page information
			pages, err := ParseArchive(path)
			if err != nil {
				log.Printf("Warning: could not parse archive %s: %v", path, err)
				return nil // Continue scanning even if one file is corrupt
			}

			// Create a new chapter model
			chapter := &models.Chapter{
				FileName:  chapterName,
				Path:      path,
				PageCount: len(pages),
				Pages:     pages,
			}

			// Add the chapter to the manga series
			manga.Chapters = append(manga.Chapters, chapter)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return mangaCollection, nil
}

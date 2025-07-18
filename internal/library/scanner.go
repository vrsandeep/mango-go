// This file contains the main logic for scanning the library directory.
// It walks the directory tree, identifies manga archives, and uses other
// helper modules to parse them and extract metadata.

package library

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"io/fs"
	"log"
	"path/filepath"

	"github.com/vrsandeep/mango-go/internal/config"
	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
)

type diskItem struct {
	path  string
	isDir bool
}

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
// func (s *Scanner) Scan(ignorePaths map[string]bool, progressChan chan<- float64) error {
// 	if progressChan != nil {
// 		defer close(progressChan)
// 	}
// 	var filesToScan []string
// 	err := filepath.WalkDir(s.cfg.Library.Path, func(path string, d os.DirEntry, err error) error {
// 		if err != nil {
// 			return err
// 		}
// 		if !d.IsDir() && (strings.HasSuffix(d.Name(), ".cbz") || strings.HasSuffix(d.Name(), ".cbr")) {
// 			filesToScan = append(filesToScan, path)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	totalFiles := len(filesToScan)
// 	for i, path := range filesToScan {
// 		if ignorePaths != nil {
// 			if _, exists := ignorePaths[path]; exists {
// 				continue // Skip already processed files in incremental scan
// 			}
// 		}
// 		err := s.processFile(path)
// 		if err != nil {
// 			return err
// 		}
// 		if progressChan != nil {
// 			progressChan <- (float64(i+1) / float64(totalFiles)) * 100
// 		}
// 	}
// 	err = s.st.UpdateAllSeriesThumbnails()
// 	if err != nil {
// 		return err
// 	}
// 	return nil

// }

// // processFile processes a single archive file, extracting metadata and updating the database.
// func (s *Scanner) processFile(path string) error {
// 	log.Printf("Processing archive: %s", path)

// 	seriesTitle, _ := ExtractMetadataFromPath(path, s.cfg.Library.Path)

// 	// Begin a transaction for this file.
// 	tx, err := s.db.Begin()
// 	if err != nil {
// 		return err
// 	}

// 	// Defer a rollback in case of error. Commit will be called on success.
// 	defer tx.Rollback()

// 	// Get or create the manga series ID.
// 	seriesID, err := s.st.GetOrCreateSeries(tx, seriesTitle, filepath.Dir(path))
// 	if err != nil {
// 		return err
// 	}

// 	// Parse the archive to get page information.
// 	pages, firstPageData, err := ParseArchive(path)
// 	if err != nil {
// 		log.Printf("Warning: could not parse archive %s: %v", path, err)
// 		return nil // Continue scanning even if one file is corrupt
// 	}

// 	var thumbnailData string
// 	if firstPageData != nil {
// 		thumbnailData, err = GenerateThumbnail(firstPageData)
// 		if err != nil {
// 			log.Printf("Warning: could not generate thumbnail for %s: %v", path, err)
// 		}
// 	}

// 	// Add or update the chapter in the database.
// 	_, err = s.st.AddOrUpdateChapter(tx, seriesID, path, len(pages), thumbnailData)
// 	if err != nil {
// 		return err
// 	}

// 	// If a thumbnail was generated, try to set it as the series cover.
// 	if thumbnailData != "" {
// 		if err := s.st.UpdateSeriesThumbnailIfNeeded(tx, seriesID, thumbnailData); err != nil {
// 			return err
// 		}
// 	}

// 	// Commit the transaction if everything was successful.
// 	return tx.Commit()
// }

// LibrarySync performs a full synchronization between the filesystem and the database.
func LibrarySync(ctx jobs.JobContext) {
	jobName := "Library Sync"
	st := store.New(ctx.DB())

	sendProgress(ctx, jobName, "Starting library sync...", 0, false)

	// 1. Preparation: Get current state from DB
	sendProgress(ctx, jobName, "Fetching current library state...", 5, false)
	dbFolders, _ := st.GetAllFoldersByPath()
	dbChapters, _ := st.GetAllChaptersByHash()

	// 2. File System Discovery
	sendProgress(ctx, jobName, "Discovering files on disk...", 10, false)
	diskItems := make(map[string]diskItem)
	rootPath := ctx.Config().Library.Path
	filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the root library folder itself
		if path == rootPath {
			return nil
		}
		diskItems[path] = diskItem{path: path, isDir: d.IsDir()}
		return nil
	})

	// 3. Reconcile Folders
	sendProgress(ctx, jobName, "Syncing folder structure...", 25, false)
	syncFolders(st, rootPath, diskItems, dbFolders)

	// Refresh folder map after sync
	dbFolders, _ = st.GetAllFoldersByPath()

	// 4. Reconcile Chapters
	sendProgress(ctx, jobName, "Syncing chapters...", 50, false)
	syncChapters(st, diskItems, dbChapters, dbFolders)

	// 5. Pruning: Remove DB entries for items no longer on disk
	sendProgress(ctx, jobName, "Pruning deleted items...", 75, false)
	prune(st, diskItems, dbFolders, dbChapters)

	// 6. Thumbnail Generation
	sendProgress(ctx, jobName, "Updating thumbnails...", 90, false)
	st.UpdateAllFolderThumbnails()

	sendProgress(ctx, jobName, "Library sync completed.", 100, true)
	log.Println("Job finished:", jobName)
}

// syncFolders ensures the folder structure in the DB matches the disk.
func syncFolders(st *store.Store, rootPath string, diskItems map[string]diskItem, dbFolders map[string]*models.Folder) {
	for path, item := range diskItems {
		if !item.isDir {
			continue
		}
		if _, exists := dbFolders[path]; !exists {
			// New folder found, create it
			parentPath := filepath.Dir(path)
			var parentID *int64
			if parent, ok := dbFolders[parentPath]; ok {
				parentID = &parent.ID
			}

			newFolder, err := st.CreateFolder(path, filepath.Base(path), parentID)
			if err != nil {
				log.Printf("Error creating folder %s: %v", path, err)
			} else {
				dbFolders[path] = newFolder // Add to map for subsequent lookups
			}
		}
	}
}

// syncChapters handles new, moved, and existing chapters.
func syncChapters(st *store.Store, diskItems map[string]diskItem, dbChapters map[string]store.ChapterInfo, dbFolders map[string]*models.Folder) {
	for path, item := range diskItems {
		if item.isDir || !IsSupportedArchive(filepath.Base(path)) { // Simplified: assume any archive is a chapter file
			continue
		}

		pages, firstPageData, err := ParseArchive(path)
		if err != nil {
			log.Printf("Could not parse archive %s: %v", path, err)
			continue
		}

		hash := generateContentHash(firstPageData, filepath.Base(path))

		if existingChapter, ok := dbChapters[hash]; ok {
			// Chapter exists, check if it moved
			if existingChapter.Path != path {
				log.Printf("Detected moved chapter: %s -> %s", existingChapter.Path, path)
				parentFolder, ok := dbFolders[filepath.Dir(path)]
				if ok {
					st.UpdateChapterPath(existingChapter.ID, path, parentFolder.ID)
				}
			}
		} else {
			// New chapter
			parentFolder, ok := dbFolders[filepath.Dir(path)]
			if ok {
				var thumb string
				if firstPageData != nil {
					thumb, _ = GenerateThumbnail(firstPageData)
				}
				st.CreateChapter(parentFolder.ID, path, hash, len(pages), thumb)
			}
		}
	}
}

// prune removes items from the DB that are no longer on disk.
func prune(st *store.Store, diskItems map[string]diskItem, dbFolders map[string]*models.Folder, dbChapters map[string]store.ChapterInfo) {
	// Prune chapters
	for hash, chapInfo := range dbChapters {
		if _, exists := diskItems[chapInfo.Path]; !exists {
			log.Printf("Pruning deleted chapter: %s", chapInfo.Path)
			st.DeleteChapterByHash(hash)
		}
	}
	// Prune folders
	for path, folder := range dbFolders {
		if _, exists := diskItems[path]; !exists {
			log.Printf("Pruning deleted folder: %s", path)
			st.DeleteFolder(folder.ID)
		}
	}
}

func generateContentHash(data []byte, filename string) string {
	hasher := sha1.New()
	hasher.Write(data)
	hasher.Write([]byte(filename))
	return hex.EncodeToString(hasher.Sum(nil))
}

func sendProgress(ctx jobs.JobContext, jobName string, message string, progress float64, done bool) {
	update := models.ProgressUpdate{
		JobName:  jobName,
		Message:  message,
		Progress: progress,
		Done:     done,
	}
	ctx.WsHub().BroadcastJSON(update)
}

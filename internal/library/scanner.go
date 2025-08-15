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
	"os"
	"path/filepath"
	"sort"
	"strings"

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



// LibrarySync performs a full synchronization between the filesystem and the database.
func LibrarySync(ctx jobs.JobContext) {
	jobId := "library-sync"
	st := store.New(ctx.DB())
	badFileStore := store.NewBadFileStore(ctx.DB())

	sendProgress(ctx, jobId, "Starting library sync...", 0, false)

	// 1. Preparation: Get current state from DB
	sendProgress(ctx, jobId, "Fetching current library state...", 5, false)
	dbFolders, _ := st.GetAllFoldersByPath()
	dbChapters, _ := st.GetAllChaptersByHash()

	// 2. File System Discovery
	sendProgress(ctx, jobId, "Discovering files on disk...", 10, false)
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
	sendProgress(ctx, jobId, "Syncing folder structure...", 25, false)
	syncFolders(st, rootPath, diskItems, dbFolders)

	// Refresh folder map after sync
	dbFolders, _ = st.GetAllFoldersByPath()

	// 4. Reconcile Chapters
	sendProgress(ctx, jobId, "Syncing chapters...", 50, false)
	parsingErrors := syncChapters(st, diskItems, dbChapters, dbFolders)

	// 5. Check for bad files during sync
	sendProgress(ctx, jobId, "Checking for bad files...", 65, false)
	checkBadFilesDuringSync(badFileStore, diskItems, parsingErrors)

	// 6. Pruning: Remove DB entries for items no longer on disk or corrupted
	sendProgress(ctx, jobId, "Pruning deleted and corrupted items...", 75, false)
	prune(st, diskItems, dbFolders, dbChapters, parsingErrors)

	// 7. Clean up bad file records for deleted files
	sendProgress(ctx, jobId, "Cleaning up bad file records...", 80, false)
	cleanupMissingBadFileRecords(badFileStore, diskItems)

	// 8. Thumbnail Generation
	sendProgress(ctx, jobId, "Updating thumbnails...", 90, false)
	st.UpdateAllFolderThumbnails()

	sendProgress(ctx, jobId, "Library sync completed.", 100, true)
	log.Println("Job finished:", jobId)
}

// syncFolders ensures the folder structure in the DB matches the disk.
func syncFolders(st *store.Store, rootPath string, diskItems map[string]diskItem, dbFolders map[string]*models.Folder) {
	// Collect all directories and sort them by path depth (shorter paths first)
	var dirPaths []string
	for path, item := range diskItems {
		if item.isDir && hasMangaArchives(path) {
			dirPaths = append(dirPaths, path)
		}
	}

	// Sort by path depth (parent folders before child folders)
	sort.Slice(dirPaths, func(i, j int) bool {
		// Count path separators to determine depth
		sepCountI := strings.Count(dirPaths[i], string(filepath.Separator))
		sepCountJ := strings.Count(dirPaths[j], string(filepath.Separator))
		if sepCountI != sepCountJ {
			return sepCountI < sepCountJ
		}
		// If same depth, sort alphabetically
		return dirPaths[i] < dirPaths[j]
	})

	// Process folders in sorted order
	for _, path := range dirPaths {
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
func syncChapters(st *store.Store, diskItems map[string]diskItem, dbChapters map[string]store.ChapterInfo, dbFolders map[string]*models.Folder) map[string]error {

	// Track parsing errors to avoid re-parsing in checkBadFilesDuringSync
	parsingErrors := make(map[string]error)

	for path, item := range diskItems {
		if item.isDir || !IsSupportedArchive(filepath.Base(path)) { // Simplified: assume any archive is a chapter file
			continue
		}

		// First check if the archive can be parsed successfully
		pages, firstPageData, err := ParseArchive(path)
		if err != nil {
			// Skip corrupted chapters - don't save them to the database
			log.Printf("Skipping corrupted archive %s: %v", path, err)
			parsingErrors[path] = err
			continue
		}

		// Only proceed with valid archives
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
			// New chapter - only create for valid archives
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

	return parsingErrors
}

// prune removes items from the DB that are no longer on disk or are corrupted.
func prune(st *store.Store, diskItems map[string]diskItem, dbFolders map[string]*models.Folder, dbChapters map[string]store.ChapterInfo, parsingErrors map[string]error) {
	// Prune chapters that are deleted or corrupted
	for hash, chapInfo := range dbChapters {
		// Check if chapter file no longer exists on disk
		if _, exists := diskItems[chapInfo.Path]; !exists {
			log.Printf("Pruning deleted chapter: %s", chapInfo.Path)
			st.DeleteChapterByHash(hash)
			continue
		}

		// Check if chapter file is corrupted
		if parseErr, isCorrupted := parsingErrors[chapInfo.Path]; isCorrupted {
			log.Printf("Pruning corrupted chapter: %s - %v", chapInfo.Path, parseErr)
			st.DeleteChapterByHash(hash)
			continue
		}
	}

	// Prune folders
	for path, folder := range dbFolders {
		if _, exists := diskItems[path]; !exists {
			log.Printf("Pruning deleted folder: %s", path)
			st.DeleteFolder(folder.ID)
		} else if diskItems[path].isDir {
			// Check if the folder still contains manga archives
			if !hasMangaArchives(path) {
				log.Printf("Pruning empty folder: %s", path)
				st.DeleteFolder(folder.ID)
			}
		}
	}
}

func generateContentHash(data []byte, filename string) string {
	hasher := sha1.New()
	hasher.Write(data)
	hasher.Write([]byte(filename))
	return hex.EncodeToString(hasher.Sum(nil))
}


// hasMangaArchives checks if a directory contains any manga archive files
func hasMangaArchives(dirPath string) bool {
	hasArchives := false
	filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip the directory itself
		if path == dirPath {
			return nil
		}
		// If we find a manga archive, mark this directory as non-empty
		if !d.IsDir() && IsSupportedArchive(d.Name()) {
			hasArchives = true
			return filepath.SkipAll // Stop walking once we find an archive
		}
		return nil
	})
	return hasArchives
}

// checkBadFilesDuringSync checks for bad files during library sync
func checkBadFilesDuringSync(badFileStore *store.BadFileStore, diskItems map[string]diskItem, parsingErrors map[string]error) {
	for path, item := range diskItems {
		if item.isDir || !IsSupportedArchive(filepath.Base(path)) {
			continue
		}

		// Check if file is accessible
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Printf("File %s is not accessible: %v", path, err)
			// Record as bad file due to I/O error
			badFileStore.CreateBadFile(path, string(models.ErrorIOError), 0)
			continue
		}

		// Check if we already know this file has parsing errors from syncChapters
		if parseErr, exists := parsingErrors[path]; exists {
			// File is corrupted or invalid
			errorMsg := categorizeError(parseErr)
			err := badFileStore.CreateBadFile(path, errorMsg, fileInfo.Size())
			if err != nil {
				log.Printf("Failed to record bad file %s: %v", path, err)
			} else {
				log.Printf("Detected bad file during sync: %s - %s", path, errorMsg)
			}
		} else {
			// File parsed successfully in syncChapters, remove from bad files if it was there
			badFileStore.DeleteBadFileByPath(path)
		}
	}
}

// cleanupMissingBadFileRecords removes bad file records for files that no longer exist on disk
func cleanupMissingBadFileRecords(badFileStore *store.BadFileStore, diskItems map[string]diskItem) {
	// Get all bad files from database
	allBadFiles, err := badFileStore.GetAllBadFiles()
	if err != nil {
		log.Printf("Error getting bad files for cleanup: %v", err)
		return
	}

	// Check each bad file record
	for _, badFile := range allBadFiles {
		// Check if the file still exists on disk
		if _, exists := diskItems[badFile.Path]; !exists {
			// File no longer exists, remove the bad file record
			log.Printf("Removing bad file record for deleted file: %s", badFile.Path)
			badFileStore.DeleteBadFile(badFile.ID)
		}
	}
}



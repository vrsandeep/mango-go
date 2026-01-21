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
	rootPath := ctx.Config().Library.Path

	sendProgress(ctx, jobId, "Starting library sync...", 0, false)

	// File System Discovery - walk entire library
	sendProgress(ctx, jobId, "Discovering files on disk...", 10, false)
	diskItems := make(map[string]diskItem)
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

	// Perform the sync with the discovered disk items
	performSync(ctx, jobId, diskItems, rootPath)

	sendProgress(ctx, jobId, "Library sync completed.", 100, true)
	log.Println("Job finished:", jobId)
}

// IncrementalLibrarySync performs an incremental scan of specific paths.
// It only scans the provided paths and their parent directories.
func IncrementalLibrarySync(ctx jobs.JobContext, changedPaths []string) error {
	jobId := "incremental-library-sync"
	rootPath := ctx.Config().Library.Path

	sendProgress(ctx, jobId, "Starting incremental library sync...", 0, false)

	// Collect all paths that need to be scanned
	// This includes the changed paths and their parent directories
	pathsToScan := make(map[string]bool)
	for _, changedPath := range changedPaths {
		// Add the changed path itself
		pathsToScan[changedPath] = true

		// Add parent directories up to the library root
		parent := filepath.Dir(changedPath)
		for parent != rootPath && parent != filepath.Dir(parent) {
			pathsToScan[parent] = true
			parent = filepath.Dir(parent)
		}
	}

	// Build diskItems map for changed paths only
	diskItems := make(map[string]diskItem)
	for path := range pathsToScan {
		// Check if path exists
		info, err := os.Stat(path)
		if err != nil {
			// Path doesn't exist (deleted), still track it for pruning
			diskItems[path] = diskItem{path: path, isDir: false}
			continue
		}

		if info.IsDir() {
			// Walk the directory to find all files
			filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if p == path {
					return nil // Skip the root directory itself
				}
				diskItems[p] = diskItem{path: p, isDir: d.IsDir()}
				return nil
			})
		} else {
			// It's a file
			diskItems[path] = diskItem{path: path, isDir: false}
		}
	}

	// Also scan the entire library root to catch any moves or deletions
	// This ensures we catch files that were moved outside the changed paths
	filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootPath {
			return nil
		}
		// Only add if not already in diskItems (to avoid duplicates)
		if _, exists := diskItems[path]; !exists {
			diskItems[path] = diskItem{path: path, isDir: d.IsDir()}
		}
		return nil
	})

	// Perform the sync with the discovered disk items
	performSync(ctx, jobId, diskItems, rootPath)

	sendProgress(ctx, jobId, "Incremental library sync completed.", 100, true)
	log.Println("Incremental library sync completed.")
	return nil
}

// performSync performs the actual synchronization work shared by both full and incremental syncs.
func performSync(ctx jobs.JobContext, jobId string, diskItems map[string]diskItem, rootPath string) {
	st := store.New(ctx.DB())
	badFileStore := store.NewBadFileStore(ctx.DB())

	// 1. Preparation: Get current state from DB
	sendProgress(ctx, jobId, "Fetching current library state...", 5, false)
	dbFolders, _ := st.GetAllFoldersByPath()
	dbChapters, _ := st.GetAllChaptersByHash()
	// Also create a map by path for quick metadata lookup
	dbChaptersByPath := make(map[string]store.ChapterInfo)
	for _, info := range dbChapters {
		dbChaptersByPath[info.Path] = info
	}

	// 2. Reconcile Folders
	sendProgress(ctx, jobId, "Syncing folder structure...", 25, false)
	syncFolders(st, rootPath, diskItems, dbFolders)

	// Refresh folder map after sync
	dbFolders, _ = st.GetAllFoldersByPath()

	// 3. Reconcile Chapters
	sendProgress(ctx, jobId, "Syncing chapters...", 50, false)
	parsingErrors := syncChapters(st, diskItems, dbChapters, dbChaptersByPath, dbFolders)

	// 4. Check for bad files during sync
	sendProgress(ctx, jobId, "Checking for bad files...", 65, false)
	checkBadFilesDuringSync(badFileStore, diskItems, parsingErrors)

	// 5. Pruning: Remove DB entries for items no longer on disk or corrupted
	sendProgress(ctx, jobId, "Pruning deleted and corrupted items...", 75, false)
	prune(st, diskItems, dbFolders, dbChapters, parsingErrors)

	// 6. Clean up bad file records for deleted files
	sendProgress(ctx, jobId, "Cleaning up bad file records...", 80, false)
	cleanupMissingBadFileRecords(badFileStore, diskItems)

	// 7. Thumbnail Generation
	sendProgress(ctx, jobId, "Updating thumbnails...", 90, false)
	st.UpdateAllFolderThumbnails()
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
// It uses file metadata (mtime, size) to skip parsing unchanged files.
func syncChapters(st *store.Store, diskItems map[string]diskItem, dbChapters map[string]store.ChapterInfo, dbChaptersByPath map[string]store.ChapterInfo, dbFolders map[string]*models.Folder) map[string]error {

	// Track parsing errors to avoid re-parsing in checkBadFilesDuringSync
	parsingErrors := make(map[string]error)
	skippedCount := 0
	parsedCount := 0

	for path, item := range diskItems {
		if item.isDir || !IsSupportedArchive(filepath.Base(path)) {
			continue
		}

		// Get file metadata before parsing
		fileInfo, err := os.Stat(path)
		if err != nil {
			log.Printf("Cannot stat file %s: %v", path, err)
			parsingErrors[path] = err
			continue
		}

		fileMtime := fileInfo.ModTime()
		fileSize := fileInfo.Size()

		// Check if we already know about this file by path
		if existingChapterByPath, existsByPath := dbChaptersByPath[path]; existsByPath {
			// File exists at this path - check if metadata changed
			if existingChapterByPath.FileMtime != nil && existingChapterByPath.FileSize != nil {
				if fileMtime.Equal(*existingChapterByPath.FileMtime) && fileSize == *existingChapterByPath.FileSize {
					// File metadata unchanged - skip parsing
					skippedCount++
					continue
				}
			}
			// Metadata changed or not set - need to re-parse
		}

		// File is new or changed - parse it
		parsedCount++
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
			// Chapter exists (by hash), check if it moved or metadata needs update
			if existingChapter.Path != path {
				log.Printf("Detected moved chapter: %s -> %s", existingChapter.Path, path)
				parentFolder, ok := dbFolders[filepath.Dir(path)]
				if ok {
					st.UpdateChapterPathWithMetadata(existingChapter.ID, path, parentFolder.ID, &fileMtime, &fileSize)
				}
			} else {
				// Same path, but metadata changed - update metadata
				parentFolder, ok := dbFolders[filepath.Dir(path)]
				if ok {
					st.UpdateChapterPathWithMetadata(existingChapter.ID, path, parentFolder.ID, &fileMtime, &fileSize)
				}
			}
		} else {
			// New chapter - create with metadata
			parentFolder, ok := dbFolders[filepath.Dir(path)]
			if ok {
				var thumb string
				if firstPageData != nil {
					thumb, _ = GenerateThumbnail(firstPageData)
				}
				st.CreateChapterWithMetadata(parentFolder.ID, path, hash, len(pages), thumb, &fileMtime, &fileSize)
			}
		}
	}

	if skippedCount > 0 {
		log.Printf("Skipped parsing %d unchanged files (metadata check)", skippedCount)
	}
	log.Printf("Parsed %d files during sync", parsedCount)

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

// This file contains the main logic for scanning the library directory.
// It walks the directory tree, identifies manga archives, and uses other
// helper modules to parse them and extract metadata.

package library

import (
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"math"
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

// RegenerateThumbnails is a new function for the admin job.
func RegenerateThumbnails(ctx jobs.JobContext) {
	jobId := "regen-thumbnails"
	sendProgress(ctx, jobId, "Regenerating thumbnails...", 0, false)
	st := store.New(ctx.DB())

	// Set the thumbnail for all chapters
	limit := 1000
	offset := 0
	totalChapters, err := st.GetTotalChaptersForThumbnailing()
	if err != nil {
		log.Printf("Error getting total chapters for thumbnails: %v", err)
	}
	for {
		chapters, err := st.GetAllChaptersForThumbnailing(limit, offset)
		if err != nil {
			log.Printf("Error updating chapters thumbnails: %v", err)
		}
		if len(chapters) == 0 {
			break
		}
		updateChaptersThumbnails(st, chapters)
		offset += limit
		progress := math.Min(float64(offset)/float64(totalChapters), 0.9) * 100
		sendProgress(ctx, jobId, fmt.Sprintf("Updating chapters thumbnails... (%d/%d)", offset, totalChapters), progress, false)
	}

	// Set the thumbnail for all folders
	sendProgress(ctx, jobId, "Updating folders thumbnails...", 90, false)
	st.UpdateAllFolderThumbnails()

	sendProgress(ctx, jobId, "Thumbnail regeneration complete.", 100, true)
}

func updateChaptersThumbnails(st *store.Store, chapters []*models.Chapter) {
	for _, chapter := range chapters {
		_, firstPageData, err := ParseArchive(chapter.Path)
		if err != nil {
			log.Printf("Error parsing archive %s: %v", chapter.Path, err)
			continue
		}
		thumbnail, err := GenerateThumbnail(firstPageData)
		if err != nil {
			log.Printf("Error generating thumbnail for chapter %d: %v", chapter.ID, err)
		}
		st.UpdateChapterThumbnail(chapter.ID, thumbnail)
	}
}

// LibrarySync performs a full synchronization between the filesystem and the database.
func LibrarySync(ctx jobs.JobContext) {
	jobId := "library-sync"
	st := store.New(ctx.DB())

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
	syncChapters(st, diskItems, dbChapters, dbFolders)

	// 5. Pruning: Remove DB entries for items no longer on disk
	sendProgress(ctx, jobId, "Pruning deleted items...", 75, false)
	prune(st, diskItems, dbFolders, dbChapters)

	// 6. Thumbnail Generation
	sendProgress(ctx, jobId, "Updating thumbnails...", 90, false)
	st.UpdateAllFolderThumbnails()

	sendProgress(ctx, jobId, "Library sync completed.", 100, true)
	log.Println("Job finished:", jobId)
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

func sendProgress(ctx jobs.JobContext, jobId string, message string, progress float64, done bool) {
	update := models.ProgressUpdate{
		JobID:    jobId,
		Message:  message,
		Progress: progress,
		Done:     done,
	}
	ctx.WsHub().BroadcastJSON(update)
}

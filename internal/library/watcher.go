// This file implements a file system watcher for incremental library scanning.
// It uses OS-level file system events to detect changes and trigger incremental scans.

package library

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/vrsandeep/mango-go/internal/jobs"
)

// WatcherService watches the library directory for file system changes
// and triggers incremental scans when files are added, modified, or deleted.
type WatcherService struct {
	ctx           jobs.JobContext
	watcher       *fsnotify.Watcher
	changedPaths  map[string]bool
	mu            sync.RWMutex
	debounceTimer *time.Timer
	debounceDelay time.Duration
	stopChan      chan struct{}
}

// NewWatcherService creates a new file system watcher service.
func NewWatcherService(ctx jobs.JobContext) *WatcherService {
	return &WatcherService{
		ctx:           ctx,
		changedPaths:  make(map[string]bool),
		debounceDelay: 2 * time.Second, // Wait 2 seconds after last change before scanning
		stopChan:      make(chan struct{}),
	}
}

// Start begins watching the library directory for changes.
func (w *WatcherService) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = watcher

	libraryPath := w.ctx.Config().Library.Path

	// Watch the library root directory recursively
	err = filepath.WalkDir(libraryPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Only watch directories (files are watched via their parent directory)
		if d.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})

	if err != nil {
		watcher.Close()
		return err
	}

	log.Printf("File watcher started for library: %s", libraryPath)

	// Start the event processing goroutine
	go w.processEvents()

	return nil
}

// Stop stops the file watcher service.
func (w *WatcherService) Stop() error {
	close(w.stopChan)
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

// processEvents processes file system events and triggers incremental scans.
func (w *WatcherService) processEvents() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)

		case <-w.stopChan:
			return
		}
	}
}

// handleEvent processes a single file system event.
func (w *WatcherService) handleEvent(event fsnotify.Event) {
	// Ignore Chmod events (these are often triggered by opening folders, reading files, etc.)
	// This prevents false triggers when browsing the file system
	if event.Op == fsnotify.Chmod {
		return
	}

	// Process Create, Write, and Remove events
	hasRelevantOp := (event.Op&fsnotify.Create == fsnotify.Create) ||
		(event.Op&fsnotify.Write == fsnotify.Write) ||
		(event.Op&fsnotify.Remove == fsnotify.Remove)

	if !hasRelevantOp {
		return
	}

	// Check if it's a directory by stat'ing it
	info, err := os.Stat(event.Name)
	isDir := err == nil && info.IsDir()

	// Handle directory creation - add to watch list and trigger scan
	if event.Op&fsnotify.Create == fsnotify.Create && isDir {
		// Add new directory to watch list
		w.watcher.Add(event.Name)
		// Trigger scan for directory creation (new folders)
		w.mu.Lock()
		w.changedPaths[event.Name] = true
		parentDir := filepath.Dir(event.Name)
		w.changedPaths[parentDir] = true
		if w.debounceTimer != nil {
			w.debounceTimer.Stop()
		}
		w.debounceTimer = time.AfterFunc(w.debounceDelay, w.triggerIncrementalScan)
		w.mu.Unlock()
		return
	}

	// For file events, only trigger on supported archive files
	if !isDir && w.isRelevantFile(event.Name) {
		w.mu.Lock()
		w.changedPaths[event.Name] = true
		// Also track parent directory
		parentDir := filepath.Dir(event.Name)
		w.changedPaths[parentDir] = true

		// Reset debounce timer
		if w.debounceTimer != nil {
			w.debounceTimer.Stop()
		}
		w.debounceTimer = time.AfterFunc(w.debounceDelay, w.triggerIncrementalScan)
		w.mu.Unlock()
	}
}

// isRelevantFile checks if a path is a relevant file (not a directory) for library scanning.
func (w *WatcherService) isRelevantFile(path string) bool {
	// Only trigger on actual archive files, not directories
	// This prevents triggering scans when folders are opened/accessed
	return IsSupportedArchive(filepath.Base(path))
}

// TriggerIncrementalScanForPath manually triggers an incremental scan for a specific path.
// This is used by the downloader when files are downloaded.
func (w *WatcherService) TriggerIncrementalScanForPath(path string) {
	w.mu.Lock()
	w.changedPaths[path] = true
	// Also track parent directory
	parentDir := filepath.Dir(path)
	w.changedPaths[parentDir] = true

	// Reset debounce timer
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.debounceTimer = time.AfterFunc(w.debounceDelay, w.triggerIncrementalScan)
	w.mu.Unlock()
}

// triggerIncrementalScan triggers an incremental scan of changed paths.
func (w *WatcherService) triggerIncrementalScan() {
	w.mu.Lock()
	if len(w.changedPaths) == 0 {
		w.mu.Unlock()
		return
	}

	// Copy changed paths and clear the map
	pathsToScan := make([]string, 0, len(w.changedPaths))
	for path := range w.changedPaths {
		pathsToScan = append(pathsToScan, path)
	}
	w.changedPaths = make(map[string]bool)
	w.mu.Unlock()

	log.Printf("File watcher detected %d changed path(s), triggering incremental scan", len(pathsToScan))

	// Run incremental scan in a goroutine to avoid blocking
	go func() {
		if err := IncrementalLibrarySync(w.ctx, pathsToScan); err != nil {
			log.Printf("Incremental scan error: %v", err)
		}
	}()
}

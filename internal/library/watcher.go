// This file implements a file system watcher for incremental library scanning.
// It uses OS-level file system events to detect changes and trigger incremental scans.

package library

import (
	"io/fs"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/vrsandeep/mango-go/internal/jobs"
)

// WatcherService watches the library directory for file system changes
// and triggers incremental scans when files are added, modified, or deleted.
type WatcherService struct {
	ctx          jobs.JobContext
	watcher      *fsnotify.Watcher
	changedPaths map[string]bool
	mu           sync.RWMutex
	debounceTimer *time.Timer
	debounceDelay time.Duration
	stopChan     chan struct{}
}

// NewWatcherService creates a new file system watcher service.
func NewWatcherService(ctx jobs.JobContext) *WatcherService {
	return &WatcherService{
		ctx:          ctx,
		changedPaths: make(map[string]bool),
		debounceDelay: 2 * time.Second, // Wait 2 seconds after last change before scanning
		stopChan:     make(chan struct{}),
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
	// Only process events for supported archive files or directories
	if !w.isRelevantPath(event.Name) {
		return
	}

	// Add the changed path to our set
	w.mu.Lock()
	w.changedPaths[event.Name] = true

	// Also track parent directory for directory events
	if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Remove == fsnotify.Remove {
		parentDir := filepath.Dir(event.Name)
		w.changedPaths[parentDir] = true

		// If a new directory was created, start watching it
		if event.Op&fsnotify.Create == fsnotify.Create {
			// Check if it's a directory by trying to add it to watcher
			// This will fail silently if it's not a directory
			w.watcher.Add(event.Name)
		}
	}

	// Reset debounce timer
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.debounceTimer = time.AfterFunc(w.debounceDelay, w.triggerIncrementalScan)
	w.mu.Unlock()
}

// isRelevantPath checks if a path is relevant for library scanning.
func (w *WatcherService) isRelevantPath(path string) bool {
	// Check if it's a supported archive file
	if IsSupportedArchive(filepath.Base(path)) {
		return true
	}
	// Check if it's a directory (we watch directories to catch new files)
	// We'll check this by seeing if the path exists and is a directory
	// For now, we'll be permissive and let the scanner decide
	return true
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

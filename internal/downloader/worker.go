package downloader

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/util"
)

var (
	jobQueue          chan *models.DownloadQueueItem
	isPaused          bool
	mu                sync.Mutex
	numWorkers        = int(math.Min(1, float64(runtime.NumCPU()))) // Number of concurrent downloads
	ErrDownloadPaused = fmt.Errorf("download paused by user")
)

// StartWorkerPool initializes and starts the download workers.
func StartWorkerPool(app *core.App) {
	jobQueue = make(chan *models.DownloadQueueItem, numWorkers)
	st := store.New(app.DB())

	// On startup, re-queue any items that were "in_progress".
	st.ResetInProgressQueueItems()

	for i := 1; i <= numWorkers; i++ {
		go worker(i, app, st)
	}

	// Start a goroutine to periodically fetch and queue jobs.
	go func() {
		for {
			mu.Lock()
			paused := isPaused
			mu.Unlock()

			if !paused {
				// Fetch enough jobs to fill the buffer if it's empty.
				// We only fetch enough jobs to fill the buffer if it's empty.
				if len(jobQueue) == 0 {
					items, err := st.GetQueuedDownloadItems(numWorkers)
					if err != nil {
						log.Printf("Error fetching queued items: %v", err)
					} else {
						for _, item := range items {
							jobQueue <- item
						}
					}
				}
			}
			time.Sleep(5 * time.Second) // Check for new jobs every 10 seconds
		}
	}()
}

func worker(id int, app *core.App, st *store.Store) {
	log.Printf("Starting download worker %d", id)
	for job := range jobQueue {
		st.UpdateQueueItemStatus(job.ID, "in_progress", "Starting download...")
		err := processDownload(app, st, job)
		if err != nil {
			// Check if this is a pause error
			if err == ErrDownloadPaused {
				log.Printf("Download paused for item %d", job.ID)
				// Don't change the status as it's already set to "paused" by the API
				continue
			}
			errMsg := fmt.Sprintf("Download failed: %v", err)
			log.Println(errMsg)
			st.UpdateQueueItemStatus(job.ID, "failed", errMsg)
		} else {
			st.UpdateQueueItemStatus(job.ID, "completed", "Download finished successfully.")
			// Trigger a library scan to pick up the new chapter
			go func() {
				app.JobManager().RunJob("library-sync", app)
			}()
		}
	}
}

func processDownload(app *core.App, st *store.Store, job *models.DownloadQueueItem) error {
	provider, ok := providers.Get(job.ProviderID)
	if !ok {
		return fmt.Errorf("provider '%s' not found", job.ProviderID)
	}

	pageURLs, err := provider.GetPageURLs(job.ChapterIdentifier)
	if err != nil {
		return fmt.Errorf("could not get page URLs: %w", err)
	}

	if len(pageURLs) == 0 {
		return fmt.Errorf("no pages found for chapter")
	}

	// Create a buffer to hold the zip archive in memory
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	total := len(pageURLs)
	for i, pageURL := range pageURLs {
		// Check if the item has been paused before starting each page download
		currentItem, err := st.GetDownloadQueueItem(job.ID)
		if err == nil && currentItem != nil && currentItem.Status == "paused" {
			log.Printf("Download paused for item %d at page %d/%d", job.ID, i+1, total)
			return ErrDownloadPaused
		}

		// Respectful delay between page downloads
		time.Sleep(250 * time.Millisecond)

		// Create HTTP request
		// Note: If the URL is a proxy URL (from plugins), it will already include
		// the necessary headers via query parameters
		req, err := http.NewRequest("GET", pageURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request for page %d: %w", i+1, err)
		}

		// Use HTTP client with timeout
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to download page %d: %w", i+1, err)
		}
		defer resp.Body.Close()

		// Check response status
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download page %d: server returned status %d", i+1, resp.StatusCode)
		}

		pageData, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read page %d data: %w", i+1, err)
		}

		// Check if page data is empty
		if len(pageData) == 0 {
			return fmt.Errorf("page %d returned empty data (may need Referer header or URL is invalid)", i+1)
		}

		// Determine file extension from Content-Type header or URL
		// Parse URL to remove query parameters before extracting extension
		var extension string
		parsedURL, err := url.Parse(pageURL)
		if err == nil {
			// Extract extension from URL path (without query parameters)
			extension = filepath.Ext(parsedURL.Path)
		} else {
			// Fallback: try to extract extension from raw URL, but remove query params
			if idx := strings.Index(pageURL, "?"); idx != -1 {
				extension = filepath.Ext(pageURL[:idx])
			} else {
				extension = filepath.Ext(pageURL)
			}
		}

		if extension == "" {
			// Try to infer from Content-Type header
			contentType := resp.Header.Get("Content-Type")
			switch {
			case strings.Contains(contentType, "image/jpeg") || strings.Contains(contentType, "image/jpg"):
				extension = ".jpg"
			case strings.Contains(contentType, "image/png"):
				extension = ".png"
			case strings.Contains(contentType, "image/gif"):
				extension = ".gif"
			case strings.Contains(contentType, "image/webp"):
				extension = ".webp"
			default:
				// Default to jpg if unknown
				extension = ".jpg"
			}
		}

		// Create a file in the zip archive
		fileName := fmt.Sprintf("page_%03d%s", i+1, extension)
		f, err := zipWriter.Create(fileName)
		if err != nil {
			return fmt.Errorf("failed to create file in zip: %w", err)
		}
		_, err = f.Write(pageData)
		if err != nil {
			return fmt.Errorf("failed to write file to zip: %w", err)
		}

		// Update progress
		progress := int((float64(i+1) / float64(total)) * 100)
		st.UpdateQueueItemProgress(job.ID, progress)

		done := progress == 100
		status := "queued"
		if progress > 0 && progress < 100 {
			status = "in_progress"
		} else if done {
			status = "completed"
		}

		// Broadcast progress update via WebSocket
		sendDownloaderProgressUpdate(app, job.ID, fmt.Sprintf("Downloaded page %d of %d", i+1, total), status, float64(progress), done)
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to finalize zip archive: %w", err)
	}

	// Save the CBZ file
	// Check if there's a subscription with a custom folder path
	var seriesDir string
	subscriptions, err := st.GetAllSubscriptions(job.ProviderID)
	if err == nil {
		for _, sub := range subscriptions {
			if sub.SeriesTitle == job.SeriesTitle && sub.FolderPath != nil {
				// Sanitize custom folder path components to ensure it's safe
				// Split the relative path and sanitize each component
				// Handle both forward and back slashes for cross-platform compatibility
				customPath := *sub.FolderPath
				// Normalize separators first
				customPath = strings.ReplaceAll(customPath, "\\", "/")
				pathComponents := strings.Split(customPath, "/")
				sanitizedComponents := make([]string, 0, len(pathComponents))
				for _, component := range pathComponents {
					if component != "" {
						sanitizedComponents = append(sanitizedComponents, util.SanitizeFolderName(component))
					}
				}
				// Rejoin sanitized components and combine with library path
				sanitizedPath := filepath.Join(sanitizedComponents...)
				seriesDir = filepath.Join(app.Config().Library.Path, sanitizedPath)
				break
			}
		}
	}

	// Fall back to default series title if no custom folder path found
	if seriesDir == "" {
		// Sanitize series title to remove invalid characters for folder names
		safeSeriesTitle := util.SanitizeFolderName(job.SeriesTitle)
		seriesDir = filepath.Join(app.Config().Library.Path, safeSeriesTitle)
	}

	if err := os.MkdirAll(seriesDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create series directory: %w", err)
	}
	// Sanitize chapter title to use as filename
	safeChapterTitle := SanitizeFilename(job.ChapterTitle)

	cbzPath := filepath.Join(seriesDir, fmt.Sprintf("%s.cbz", safeChapterTitle))
	if err := os.WriteFile(cbzPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to save CBZ file: %w", err)
	}

	return nil
}

func SanitizeFilename(filename string) string {
	re := regexp.MustCompile(`[\x00\\/:*?"<>|]`)
	safeChapterTitle := re.ReplaceAllString(filename, "-")
	safeChapterTitle = strings.ReplaceAll(safeChapterTitle, "\x00", "-")

	for strings.HasPrefix(safeChapterTitle, ".") || strings.HasPrefix(safeChapterTitle, "-") {
		safeChapterTitle = safeChapterTitle[1:]
	}
	if safeChapterTitle == "" {
		safeChapterTitle = "untitled"
	}
	return safeChapterTitle
}

// Control functions for the download queue
func PauseDownloads() { mu.Lock(); isPaused = true; mu.Unlock(); log.Println("Download queue paused.") }
func ResumeDownloads() {
	mu.Lock()
	isPaused = false
	mu.Unlock()
	log.Println("Download queue resumed.")
}
func IsPaused() bool { mu.Lock(); defer mu.Unlock(); return isPaused }

// PauseQueueItem pauses a specific item and broadcasts the status change
func PauseQueueItem(app *core.App, st *store.Store, itemID int64) error {
	err := st.PauseQueueItem(itemID)
	if err != nil {
		return err
	}

	// Get current item to preserve progress
	currentItem, getErr := st.GetDownloadQueueItem(itemID)
	progress := 0.0
	if getErr == nil && currentItem != nil {
		progress = float64(currentItem.Progress)
	}

	// Broadcast pause status update
	sendDownloaderProgressUpdate(app, itemID, "Download paused by user", "paused", progress, false)

	return nil
}

// ResumeQueueItem resumes a specific item and broadcasts the status change
func ResumeQueueItem(app *core.App, st *store.Store, itemID int64) error {
	err := st.ResumeQueueItem(itemID)
	if err != nil {
		return err
	}

	// Get current item to preserve progress
	currentItem, getErr := st.GetDownloadQueueItem(itemID)
	progress := 0.0
	if getErr == nil && currentItem != nil {
		progress = float64(currentItem.Progress)
	}

	// Broadcast resume status update
	sendDownloaderProgressUpdate(app, itemID, "Download resumed by user", "queued", progress, false)

	return nil
}

func sendDownloaderProgressUpdate(app *core.App, itemID int64, message string, status string, progress float64, done bool) {
	app.WsHub().BroadcastJSON(models.ProgressUpdate{
		JobID:    "downloader",
		Message:  message,
		Progress: progress,
		ItemID:   itemID,
		Status:   status,
		Done:     done,
	})
}

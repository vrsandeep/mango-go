package downloader

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/downloader/providers"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/websocket"
)

var (
	jobQueue   chan *models.DownloadQueueItem
	isPaused   bool
	mu         sync.Mutex
	numWorkers = int(math.Min(4, float64(runtime.NumCPU()))) // Number of concurrent downloads
)

// StartWorkerPool initializes and starts the download workers.
func StartWorkerPool(app *core.App) {
	jobQueue = make(chan *models.DownloadQueueItem, numWorkers)
	st := store.New(app.DB)

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
			errMsg := fmt.Sprintf("Download failed: %v", err)
			log.Println(errMsg)
			st.UpdateQueueItemStatus(job.ID, "failed", errMsg)
		} else {
			st.UpdateQueueItemStatus(job.ID, "completed", "Download finished successfully.")
			// Trigger a library scan to pick up the new chapter
			go func() {
				scanner := library.NewScanner(app.Config, app.DB)
				scanner.Scan(nil, nil)
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
		// Respectful delay between page downloads
		time.Sleep(250 * time.Millisecond)

		resp, err := http.Get(pageURL)
		if err != nil {
			return fmt.Errorf("failed to download page %d: %w", i+1, err)
		}
		defer resp.Body.Close()

		pageData, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read page %d data: %w", i+1, err)
		}

		// Create a file in the zip archive
		fileName := fmt.Sprintf("page_%03d%s", i+1, filepath.Ext(pageURL))
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
		app.WsHub.BroadcastJSON(websocket.ProgressUpdate{
			JobName:  "downloader",
			Message:  fmt.Sprintf("Downloaded page %d of %d", i+1, total),
			Progress: float64(progress),
			ItemID:   job.ID,
			Status:   status,
			Done:     done,
		})
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to finalize zip archive: %w", err)
	}

	// Save the CBZ file
	seriesDir := filepath.Join(app.Config.Library.Path, job.SeriesTitle)
	if err := os.MkdirAll(seriesDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create series directory: %w", err)
	}
	// Sanitize chapter title to use as filename
	safeChapterTitle := strings.ReplaceAll(job.ChapterTitle, "/", "-")
	cbzPath := filepath.Join(seriesDir, fmt.Sprintf("%s.cbz", safeChapterTitle))
	if err := os.WriteFile(cbzPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to save CBZ file: %w", err)
	}

	return nil
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

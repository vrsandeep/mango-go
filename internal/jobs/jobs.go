package jobs

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/vrsandeep/mango-go/internal/core"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/store"
)

type ProgressUpdate struct {
	JobName  string  `json:"job_name"`
	Message  string  `json:"message"`
	Progress float64 `json:"progress"`
	Done     bool    `json:"done"`
}

func sendProgress(app *core.App, jobName, message string, progress float64, done bool) {
	update := ProgressUpdate{
		JobName:  jobName,
		Message:  message,
		Progress: progress,
		Done:     done,
	}
	app.WsHub.BroadcastJSON(update)
}

func RunFullScan(app *core.App) {
	jobName := "Full Scan"
	sendProgress(app, jobName, "Starting full library scan...", 0, false)
	log.Println("Job started:", jobName)

	scanner := library.NewScanner(app.Config, app.DB)

	// Create a channel to receive progress updates from the scanner
	progressChan := make(chan float64)
	go func() {
		for p := range progressChan {
			sendProgress(app, jobName, fmt.Sprintf("Scanning... %.0f%%", p), p*0.9, false) // Scale progress from 0 to 90
		}
	}()

	if err := scanner.Scan(nil, progressChan); err != nil {
		errMsg := fmt.Sprintf("Error during scan: %v", err)
		sendProgress(app, jobName, errMsg, 100, true)
		log.Println(errMsg)
		return
	}

	sendProgress(app, jobName, "Full scan completed successfully.", 100, true)
	log.Println("Job finished:", jobName)
}

func RunIncrementalScan(app *core.App) {
	jobName := "Incremental Scan"
	sendProgress(app, jobName, "Finding existing chapters...", 0, false)
	log.Println("Job started:", jobName)

	st := store.New(app.DB)
	existingPaths, err := st.GetAllChapterPaths()
	if err != nil {
		errMsg := fmt.Sprintf("Error getting existing paths: %v", err)
		sendProgress(app, jobName, errMsg, 100, true)
		log.Println(errMsg)
		return
	}
	pathSet := make(map[string]bool)
	for _, path := range existingPaths {
		pathSet[path] = true
	}

	sendProgress(app, jobName, "Scanning library for new chapters...", 10, false)
	scanner := library.NewScanner(app.Config, app.DB)

	// Create a channel to receive progress updates from the scanner
	progressChan := make(chan float64)
	go func() {
		for p := range progressChan {
			sendProgress(app, jobName, fmt.Sprintf("Scanning... %.0f%%", p), 10+p*0.9, false) // Scale progress from 10 to 100
		}
	}()

	scanner.Scan(pathSet, progressChan)

	sendProgress(app, jobName, "Incremental scan completed.", 100, true)
	log.Println("Job finished:", jobName)
}

func RunPruneDatabase(app *core.App) {
	jobName := "Prune Database"
	sendProgress(app, jobName, "Getting all database entries...", 0, false)
	log.Println("Job started:", jobName)

	st := store.New(app.DB)
	allPaths, err := st.GetAllChapterPaths()
	if err != nil {
		errMsg := fmt.Sprintf("Error getting chapter paths: %v", err)
		sendProgress(app, jobName, errMsg, 100, true)
		log.Println(errMsg)
		return
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var deletedCount int
	var processedCount int
	total := len(allPaths)
	if total == 0 {
		sendProgress(app, jobName, "Pruning complete. No chapters to check.", 100, true)
		log.Println("Job finished:", jobName)
		return
	}

	for i, path := range allPaths {
		wg.Add(1)
		go func(p string, idx int) {
			defer wg.Done()
			if _, err := os.Stat(p); os.IsNotExist(err) {
				mu.Lock()
				deletedCount++
				mu.Unlock()
				if err := st.DeleteChapterByPath(p); err != nil {
					log.Printf("Failed to prune chapter %s: %v", p, err)
				}
			}
			mu.Lock()
			processedCount++
			mu.Unlock()
			progress := (float64(processedCount) / float64(total)) * 100
			// Update progress periodically or on the last item
			if processedCount%20 == 0 || processedCount == total-1 {
				mu.Lock()
				currentDeleted := deletedCount
				mu.Unlock()
				sendProgress(app, jobName, fmt.Sprintf("Checking... (%d/%d) | Deleted: %d", idx+1, total, currentDeleted), progress, false)
			}
		}(path, i)
	}
	wg.Wait()

	sendProgress(app, jobName, "Deleting empty series...", 99, false)
	st.DeleteEmptySeries()

	if deletedCount == 0 {
		sendProgress(app, jobName, "Pruning complete. No non-existent chapters found.", 100, true)
	} else {
		finalMsg := fmt.Sprintf("Pruning complete. Removed %d non-existent chapter(s).", deletedCount)
		sendProgress(app, jobName, finalMsg, 100, true)
	}
	log.Println("Job finished:", jobName)
}

func RunThumbnailGeneration(app *core.App) {
	jobName := "Regenerate Thumbnails"
	sendProgress(app, jobName, "Getting all chapters...", 0, false)
	log.Println("Job started:", jobName)

	st := store.New(app.DB)
	allChapters, err := st.GetAllChaptersForThumbnailing()
	if err != nil {
		errMsg := fmt.Sprintf("Error getting chapters: %v", err)
		sendProgress(app, jobName, errMsg, 100, true)
		log.Println(errMsg)
		return
	}
	total := len(allChapters)
	if total == 0 {
		sendProgress(app, jobName, "Thumbnail generation complete. No chapters found.", 100, true)
		log.Println("Job finished:", jobName)
		return
	}

	for i, ch := range allChapters {
		_, firstPageData, err := library.ParseArchive(ch.Path)
		if err == nil && firstPageData != nil {
			thumbnail, thumbErr := library.GenerateThumbnail(firstPageData)
			if thumbErr == nil {
				st.UpdateChapterThumbnail(ch.ID, thumbnail)
			}
		}
		progress := (float64(i+1) / float64(total)) * 100
		if i%20 == 0 || i == total-1 {
			sendProgress(app, jobName, fmt.Sprintf("Generating... (%d/%d)", i+1, total), progress, false)
		}
	}

	sendProgress(app, jobName, "Updating series covers...", 99, false)
	st.UpdateAllSeriesThumbnails()

	sendProgress(app, jobName, "Thumbnail regeneration complete.", 100, true)
	log.Println("Job finished:", jobName)
}

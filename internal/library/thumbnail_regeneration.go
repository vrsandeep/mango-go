// This file contains thumbnail regeneration functionality for the library.

package library

import (
	"fmt"
	"log"
	"math"
	"path/filepath"

	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
)

// RegenerateThumbnails regenerates thumbnails for all chapters and folders.
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
		updateChaptersThumbnails(ctx, jobId, st, chapters, offset, totalChapters)
		offset += limit
	}

	// Set the thumbnail for all folders
	sendProgress(ctx, jobId, "Updating folders thumbnails...", 90, false)
	st.UpdateAllFolderThumbnails()

	sendProgress(ctx, jobId, "Thumbnail regeneration complete.", 100, true)
}

// updateChaptersThumbnails updates thumbnails for a batch of chapters.
func updateChaptersThumbnails(
	ctx jobs.JobContext,
	jobId string,
	st *store.Store,
	chapters []*models.Chapter,
	offset,
	totalChapters int,
) {
	for i, chapter := range chapters {
		_, firstPageData, err := ParseArchive(chapter.Path)
		if err != nil {
			log.Printf("Error parsing archive %s: %v %v", chapter.Path, err, chapter)
			continue
		}
		thumbnail, err := GenerateThumbnail(firstPageData)
		if err != nil {
			log.Printf("Error generating thumbnail for chapter %d: %v", chapter.ID, err)
		}
		st.UpdateChapterThumbnail(chapter.ID, thumbnail)

		// Calculate and send progress for each individual file
		currentProgress := offset + i + 1
		progress := math.Min(float64(currentProgress)/float64(totalChapters), 0.9) * 100
		sendProgress(ctx, jobId, fmt.Sprintf("Updating chapter thumbnail %d/%d: %s", currentProgress, totalChapters, filepath.Base(chapter.Path)), progress, false)
	}
}

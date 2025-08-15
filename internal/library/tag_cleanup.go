// This file contains tag cleanup functionality for the library.

package library

import (
	"log"

	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/store"
)

// DeleteEmptyTags removes tags that are no longer associated with any chapters.
func DeleteEmptyTags(ctx jobs.JobContext) {
	jobId := "delete-empty-tags"
	sendProgress(ctx, jobId, "Deleting empty tags...", 0, false)
	st := store.New(ctx.DB())

	err := st.DeleteEmptyTags()
	if err != nil {
		log.Printf("Error deleting empty tags: %v", err)
	}

	sendProgress(ctx, jobId, "Deleting empty tags...", 100, true)
}

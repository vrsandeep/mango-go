// This file handles detection of bad or unreadable chapter files in the library.

package library

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/library/chapterfiles"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
)

// DetectBadFiles scans the library for chapter files that fail inspection (corrupt or invalid).
func DetectBadFiles(ctx jobs.JobContext) {
	jobId := "detect-bad-files"
	sendProgress(ctx, jobId, "Starting bad file detection...", 0, false)

	badFileStore := store.NewBadFileStore(ctx.DB())

	rootPath := ctx.Config().Library.Path
	sendProgress(ctx, jobId, "Scanning library for chapter files...", 10, false)

	var chapterFilePaths []string
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && chapterfiles.IsSupportedChapterFile(d.Name()) {
			chapterFilePaths = append(chapterFilePaths, path)
		}
		return nil
	})

	if err != nil {
		log.Printf("Error walking library directory: %v", err)
		sendProgress(ctx, jobId, "Error scanning library directory", 0, true)
		return
	}

	totalFiles := len(chapterFilePaths)
	if totalFiles == 0 {
		sendProgress(ctx, jobId, "No chapters found in library", 100, true)
		return
	}

	sendProgress(ctx, jobId, fmt.Sprintf("Found %d chapters, checking for problems...", totalFiles), 20, false)

	// Check each chapter file
	badFileCount := 0
	for i, filePath := range chapterFilePaths {
		progress := 20 + (float64(i) / float64(totalFiles) * 70)
		sendProgress(ctx, jobId, fmt.Sprintf("Checking file %d/%d: %s", i+1, totalFiles, filepath.Base(filePath)), progress, false)

		// Check if file is accessible
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			// File doesn't exist or can't be accessed
			log.Printf("File %s is not accessible: %v", filePath, err)
			continue
		}

		// Try to parse the chapter file
		_, _, parseErr := chapterfiles.InspectChapterFile(context.Background(), filePath)
		if parseErr != nil {
			// File is corrupted or invalid
			errorMsg := categorizeError(parseErr)
			err := badFileStore.CreateBadFile(filePath, errorMsg, fileInfo.Size())
			if err != nil {
				log.Printf("Failed to record bad file %s: %v", filePath, err)
			} else {
				badFileCount++
				log.Printf("Detected bad file: %s - %s", filePath, errorMsg)
			}
		} else {
			// File is good, remove it from bad files list if it was there
			badFileStore.DeleteBadFileByPath(filePath)
		}
	}

	// Final progress update
	if badFileCount == 0 {
		sendProgress(ctx, jobId, "No bad files detected. All chapters look valid.", 100, true)
	} else {
		sendProgress(ctx, jobId, fmt.Sprintf("Detection complete. Found %d bad files.", badFileCount), 100, true)
	}
}

// categorizeError maps parser errors to stable BadFileError values stored in the DB.
func categorizeError(err error) string {
	errorStr := err.Error()

	// Check for specific error patterns
	if contains(errorStr, "zip: not a valid zip file") || contains(errorStr, "archive/zip: not a valid zip file") {
		return string(models.ErrorCorruptedChapterFile)
	}
	if contains(errorStr, "unsupported archive type") ||
		strings.Contains(errorStr, "unsupported chapter file") ||
		strings.Contains(errorStr, "unsupported archive ") {
		return string(models.ErrorUnsupportedFormat)
	}
	if contains(errorStr, "no image files found") || contains(errorStr, "no pages found") {
		return string(models.ErrorEmptyChapterFile)
	}
	if contains(errorStr, "failed to open") || contains(errorStr, "permission denied") {
		return string(models.ErrorIOError)
	}
	if contains(errorStr, "password") || contains(errorStr, "encrypted") {
		return string(models.ErrorPasswordProtected)
	}

	// Default to invalid format for unknown errors
	return string(models.ErrorInvalidFormat)
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(s) == len(substr) ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr))))
}

// containsSubstring is a simple substring search
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

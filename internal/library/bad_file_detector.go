// This file handles detection of bad/corrupted archive files in the library.

package library

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
)

// DetectBadFiles scans the library for corrupted or invalid archive files.
func DetectBadFiles(ctx jobs.JobContext) {
	jobId := "detect-bad-files"
	sendProgress(ctx, jobId, "Starting bad file detection...", 0, false)

	badFileStore := store.NewBadFileStore(ctx.DB())

	rootPath := ctx.Config().Library.Path
	sendProgress(ctx, jobId, "Scanning library for bad files...", 10, false)

	// Get all archive files in the library
	var archiveFiles []string
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && IsSupportedArchive(d.Name()) {
			archiveFiles = append(archiveFiles, path)
		}
		return nil
	})

	if err != nil {
		log.Printf("Error walking library directory: %v", err)
		sendProgress(ctx, jobId, "Error scanning library directory", 0, true)
		return
	}

	totalFiles := len(archiveFiles)
	if totalFiles == 0 {
		sendProgress(ctx, jobId, "No archive files found in library", 100, true)
		return
	}

	sendProgress(ctx, jobId, fmt.Sprintf("Found %d archive files, checking for corruption...", totalFiles), 20, false)

	// Check each archive file
	badFileCount := 0
	for i, filePath := range archiveFiles {
		progress := 20 + (float64(i) / float64(totalFiles) * 70)
		sendProgress(ctx, jobId, fmt.Sprintf("Checking file %d/%d: %s", i+1, totalFiles, filepath.Base(filePath)), progress, false)

		// Check if file is accessible
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			// File doesn't exist or can't be accessed
			log.Printf("File %s is not accessible: %v", filePath, err)
			continue
		}

		// Try to parse the archive
		_, _, parseErr := ParseArchive(filePath)
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
		sendProgress(ctx, jobId, "No bad files detected. All archives are valid.", 100, true)
	} else {
		sendProgress(ctx, jobId, fmt.Sprintf("Detection complete. Found %d bad files.", badFileCount), 100, true)
	}
}

// categorizeError categorizes parsing errors into user-friendly categories
func categorizeError(err error) string {
	errorStr := err.Error()

	// Check for specific error patterns
	if contains(errorStr, "zip: not a valid zip file") || contains(errorStr, "archive/zip: not a valid zip file") {
		return string(models.ErrorCorruptedArchive)
	}
	if contains(errorStr, "unsupported archive type") {
		return string(models.ErrorUnsupportedFormat)
	}
	if contains(errorStr, "no image files found") {
		return string(models.ErrorEmptyArchive)
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

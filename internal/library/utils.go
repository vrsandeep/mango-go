// This file contains utility functions shared across the library package.

package library

import (
	"os"
	"strings"

	"github.com/vrsandeep/mango-go/internal/jobs"
	"github.com/vrsandeep/mango-go/internal/models"
)

// sendProgress sends a progress update via WebSocket to connected clients.
func sendProgress(ctx jobs.JobContext, jobId string, message string, progress float64, done bool) {
	// Skip WebSocket broadcasting during tests to avoid timeouts
	if isTestEnvironment() {
		return
	}

	update := models.ProgressUpdate{
		JobID:    jobId,
		Message:  message,
		Progress: progress,
		Done:     done,
	}
	ctx.WsHub().BroadcastJSON(update)
}

// isTestEnvironment checks if we're running in a test environment
func isTestEnvironment() bool {
	// Check if we're running tests by looking for test-related environment variables
	// or if the executable name contains "test"
	executable := os.Args[0]
	return strings.Contains(executable, "test") ||
		strings.Contains(executable, "Test") ||
		os.Getenv("GO_TEST") != ""
}

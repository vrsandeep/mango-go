package testutil

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// CreateTestCBZ is a helper function that creates a temporary CBZ file with
// a given set of page names. It's useful for testing archive parsing.
func CreateTestCBZ(t *testing.T, dir, name string, pages []string) string {
	t.Helper()
	filePath := filepath.Join(dir, name)
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("Failed to create temp cbz file: %v", err)
	}
	t.Cleanup(func() { file.Close() }) // Ensure file is closed after test

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	for _, page := range pages {
		_, err := zipWriter.Create(page)
		if err != nil {
			t.Fatalf("Failed to create entry '%s' in zip: %v", page, err)
		}
	}
	return filePath
}

// This file tests the archive parsing logic for CBZ and CBR files.
// It includes a helper function to create a temporary test CBZ file.

package library

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// createTestCBZ is a helper function that creates a temporary CBZ file
// for testing purposes. It returns the path to the created file.
func createTestCBZ(t *testing.T, dir string) string {
	t.Helper()

	// Create a temporary file
	file, err := os.Create(filepath.Join(dir, "test.cbz"))
	if err != nil {
		t.Fatalf("Failed to create temp cbz file: %v", err)
	}
	defer file.Close()

	// Create a new zip archive
	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Add files to the archive
	files := []struct {
		Name    string
		Content string
	}{
		{"01.jpg", "image data"},
		{"03.png", "image data"},
		{"02.jpeg", "image data"},
		{"notes.txt", "some text"},
		{"nested/", ""}, // A directory
	}

	for _, f := range files {
		writer, err := zipWriter.Create(f.Name)
		if err != nil {
			t.Fatalf("Failed to create entry in zip: %v", err)
		}
		if f.Content != "" {
			_, err = writer.Write([]byte(f.Content))
			if err != nil {
				t.Fatalf("Failed to write to zip entry: %v", err)
			}
		}
	}

	return file.Name()
}

func TestParseArchive(t *testing.T) {
	// Create a temporary directory for our test files
	tempDir := t.TempDir()

	t.Run("Parse Valid CBZ", func(t *testing.T) {
		cbzPath := createTestCBZ(t, tempDir)

		pages, err := ParseArchive(cbzPath)
		if err != nil {
			t.Fatalf("ParseArchive failed for CBZ: %v", err)
		}

		// Check page count - should only include images
		if len(pages) != 3 {
			t.Errorf("Expected 3 pages, but got %d", len(pages))
		}

		// Check sorting order
		if pages[0].FileName != "01.jpg" || pages[1].FileName != "02.jpeg" || pages[2].FileName != "03.png" {
			t.Errorf("Pages are not sorted correctly: got %s, %s, %s", pages[0].FileName, pages[1].FileName, pages[2].FileName)
		}

		// Check indices
		if pages[0].Index != 0 || pages[1].Index != 1 || pages[2].Index != 2 {
			t.Errorf("Page indices are not set correctly")
		}
	})

	t.Run("Parse CBR (Not Implemented)", func(t *testing.T) {
		cbrPath := filepath.Join(tempDir, "test.cbr")
		os.WriteFile(cbrPath, []byte{}, 0644)

		pages, err := ParseArchive(cbrPath)
		if err != nil {
			t.Fatalf("ParseArchive failed for CBR: %v", err)
		}
		if len(pages) != 0 {
			t.Errorf("Expected 0 pages for unimplemented CBR, got %d", len(pages))
		}
	})

	t.Run("Unsupported Archive Type", func(t *testing.T) {
		unsupportedPath := filepath.Join(tempDir, "test.txt")
		os.WriteFile(unsupportedPath, []byte("hello"), 0644)

		_, err := ParseArchive(unsupportedPath)
		if err == nil {
			t.Error("Expected an error for unsupported archive type, but got nil")
		}
	})
}

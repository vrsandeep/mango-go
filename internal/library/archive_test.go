// This file tests the archive parsing logic for CBZ and CBR files.
// It includes a helper function to create a temporary test CBZ file.

package library_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrsandeep/mango-go/internal/library"
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

		pages, firstPageData, err := library.ParseArchive(cbzPath)
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

		if firstPageData == nil {
			t.Error("Expected first page data to be non-nil")
		}
	})

	t.Run("Unsupported Archive Type", func(t *testing.T) {
		unsupportedPath := filepath.Join(tempDir, "test.txt")
		os.WriteFile(unsupportedPath, []byte("hello"), 0644)

		_, _, err := library.ParseArchive(unsupportedPath)
		if err == nil {
			t.Error("Expected an error for unsupported archive type, but got nil")
		}
	})
}

func TestParseCBR(t *testing.T) {

	// The file contains three images and one directory.
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	// Change to project root (which is two levels up from internal/api)
	if err := os.Chdir("../"); err != nil {
		t.Fatalf("Failed to change directory to project root: %v", err)
	}
	// Ensure we change back to the original directory after the test
	defer os.Chdir(originalWD)

	assetDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get asset directory: %v", err)
	}
	cbrPath := filepath.Join(assetDir, "testutil", "asset", "test.cbr")
	// print cbrPath for debugging
	t.Logf("Testing CBR parsing with file: %s", cbrPath)

	pages, firstPageData, err := library.ParseArchive(cbrPath)
	if err != nil {
		t.Fatalf("ParseArchive failed for CBR: %v", err)
	}
	if len(pages) != 4 {
		t.Errorf("Expected 4 pages for unimplemented CBR, got %d", len(pages))
	}
	if firstPageData == nil {
		t.Error("Expected first page data to be not nil")
	}
}

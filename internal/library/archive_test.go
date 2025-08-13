// This file tests the archive parsing logic for CBZ and CBR files.
// It includes a helper function to create a temporary test CBZ file.

package library_test

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// createTestCBZWithContent creates a CBZ file with specific content for testing
func createTestCBZWithContent(t *testing.T, dir, filename string, files []struct {
	Name    string
	Content string
}) string {
	t.Helper()

	file, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("Failed to create temp cbz file: %v", err)
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

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

// Test that archive type detection works correctly through ParseArchive
func TestArchiveTypeDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files for different archive types
	testCases := []struct {
		name          string
		filename      string
		shouldSucceed bool
	}{
		{"CBZ file", "test.cbz", true},
		{"ZIP file", "archive.zip", true},
		{"CBR file", "test.cbr", true},
		{"RAR file", "archive.rar", true},
		{"7Z file", "archive.7z", true},
		{"CB7 file", "test.cb7", true},
		{"Unknown extension", "test.txt", false},
		{"No extension", "filename", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldSucceed {
				// For supported types, create a minimal valid archive
				if strings.HasSuffix(tc.filename, ".cbz") || strings.HasSuffix(tc.filename, ".zip") {
					// Create a minimal ZIP archive with at least one image file
					file, err := os.Create(filepath.Join(tempDir, tc.filename))
					if err != nil {
						t.Fatalf("Failed to create test file: %v", err)
					}
					defer file.Close()

					zipWriter := zip.NewWriter(file)

					// Add a minimal image file to make the archive valid
					writer, err := zipWriter.Create("test.jpg")
					if err != nil {
						t.Fatalf("Failed to create zip entry: %v", err)
					}
					writer.Write([]byte("fake image data"))

					zipWriter.Close()

					_, _, err = library.ParseArchive(file.Name())
					if err != nil {
						t.Errorf("Expected ParseArchive to succeed for %s, got error: %v", tc.filename, err)
					}
				} else {
					// For other archive types, just test that they're recognized as supported
					// We can't easily create test files for these formats
					t.Skipf("Skipping test for %s - cannot easily create test file", tc.filename)
				}
			} else {
				// For unsupported types, create a simple text file
				unsupportedPath := filepath.Join(tempDir, tc.filename)
				if err := os.WriteFile(unsupportedPath, []byte("content"), 0644); err != nil {
					t.Fatalf("Failed to create unsupported file: %v", err)
				}

				_, _, err := library.ParseArchive(unsupportedPath)
				if err == nil {
					t.Errorf("Expected ParseArchive to fail for %s, but got no error", tc.filename)
				}
			}
		})
	}
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

func TestGetPageFromArchive(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test CBZ with multiple pages
	files := []struct {
		Name    string
		Content string
	}{
		{"01.jpg", "page 1 content"},
		{"02.png", "page 2 content"},
		{"03.jpeg", "page 3 content"},
	}
	defer os.RemoveAll(tempDir)

	cbzPath := createTestCBZWithContent(t, tempDir, "test.cbz", files)

	t.Run("Valid page index", func(t *testing.T) {
		// Test getting first page (index 0)
		data, filename, err := library.GetPageFromArchive(cbzPath, 0)
		if err != nil {
			t.Fatalf("GetPageFromArchive failed: %v", err)
		}

		if filename != "01.jpg" {
			t.Errorf("Expected filename 01.jpg, got %s", filename)
		}

		if string(data) != "page 1 content" {
			t.Errorf("Expected content 'page 1 content', got %q", string(data))
		}

		// Test getting second page (index 1)
		data, filename, err = library.GetPageFromArchive(cbzPath, 1)
		if err != nil {
			t.Fatalf("GetPageFromArchive failed for page 1: %v", err)
		}

		if filename != "02.png" {
			t.Errorf("Expected filename 02.png, got %s", filename)
		}

		if string(data) != "page 2 content" {
			t.Errorf("Expected content 'page 2 content', got %q", string(data))
		}
	})

	t.Run("Invalid page index", func(t *testing.T) {
		// Test negative index
		_, _, err := library.GetPageFromArchive(cbzPath, -1)
		if err == nil {
			t.Error("Expected error for negative page index, got nil")
		}

		// Test out of bounds index
		_, _, err = library.GetPageFromArchive(cbzPath, 10)
		if err == nil {
			t.Error("Expected error for out of bounds page index, got nil")
		}
	})

	t.Run("Unsupported archive type", func(t *testing.T) {
		// Create an unsupported file
		unsupportedPath := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(unsupportedPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create unsupported file: %v", err)
		}

		_, _, err := library.GetPageFromArchive(unsupportedPath, 0)
		if err == nil {
			t.Error("Expected error for unsupported archive type, got nil")
		}
	})
}

// Test that the refactored functions maintain the same behavior
func TestWeirdFileOrdering(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Page sorting and indexing consistency", func(t *testing.T) {
		// Create a CBZ with unsorted filenames
		files := []struct {
			Name    string
			Content string
		}{
			{"003.png", "page 2"},
			{"02.jpg", "page 1"},
			{"04.jpeg", "page 3"},
			{"__1.jpg", "page 4"},
		}

		cbzPath := createTestCBZWithContent(t, tempDir, "unsorted.cbz", files)

		// Parse the archive
		pages, _, err := library.ParseArchive(cbzPath)
		if err != nil {
			t.Fatalf("ParseArchive failed: %v", err)
		}

		// Verify pages are sorted and indexed correctly
		expectedOrder := []string{"02.jpg", "003.png", "04.jpeg", "__1.jpg"}
		for i, page := range pages {
			if page.FileName != expectedOrder[i] {
				t.Errorf("Page[%d] filename = %s, want %s", i, page.FileName, expectedOrder[i])
			}
			if page.Index != i {
				t.Errorf("Page[%d] index = %d, want %d", i, page.Index, i)
			}
		}

		// Test that GetPageFromArchive returns pages in the same order
		for i, expectedFilename := range expectedOrder {
			data, filename, err := library.GetPageFromArchive(cbzPath, i)
			if err != nil {
				t.Fatalf("GetPageFromArchive failed for page %d: %v", i, err)
			}

			if filename != expectedFilename {
				t.Errorf("GetPageFromArchive(%d) returned filename %s, want %s", i, filename, expectedFilename)
			}

			if string(data) != fmt.Sprintf("page %d", i+1) {
				t.Errorf("GetPageFromArchive(%d) returned wrong content: %s", i, string(data))
			}
		}
	})
}

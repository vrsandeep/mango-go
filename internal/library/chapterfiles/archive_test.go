// Tests for the archive chapter-file handler (CBZ, CBR, zip, rar, 7z).

package chapterfiles_test

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrsandeep/mango-go/internal/library/chapterfiles"
)

// createTestCBZ creates a temporary CBZ file for testing. It returns the path to the created file.
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

// TestSupportedExtensions verifies InspectChapterFile for supported vs unsupported basenames.
func TestArchiveTypeDetection(t *testing.T) {
	tempDir := t.TempDir()

	// Supported extensions use real zip payloads where needed; others are skipped (hard to synthesize).
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

	ctx := context.Background()
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

					_, _, err = chapterfiles.InspectChapterFile(ctx, file.Name())
					if err != nil {
						t.Errorf("Expected InspectChapterFile to succeed for %s, got error: %v", tc.filename, err)
					}
				} else {
					// For other archive types, just test that they're recognized as supported
					// We can't easily create test files for these formats
					t.Skipf("Skipping test for %s - cannot easily create test file", tc.filename)
				}
			} else {
				// For unsupported types, create a file with content and test that it's recognized as unsupported
				unsupportedPath := filepath.Join(tempDir, tc.filename)
				if err := os.WriteFile(unsupportedPath, []byte("content"), 0644); err != nil {
					t.Fatalf("Failed to create unsupported file: %v", err)
				}

				_, _, err := chapterfiles.InspectChapterFile(ctx, unsupportedPath)
				if err == nil {
					t.Errorf("Expected InspectChapterFile to fail for %s, but got no error", tc.filename)
				}
			}
		})
	}
}

func TestInspectChapterFile_CBZ(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	t.Run("Parse Valid CBZ", func(t *testing.T) {
		cbzPath := createTestCBZ(t, tempDir)

		pages, firstPageData, err := chapterfiles.InspectChapterFile(ctx, cbzPath)
		if err != nil {
			t.Fatalf("InspectChapterFile failed for CBZ: %v", err)
		}

		// Check page count - should only include images
		if len(pages) != 3 {
			t.Errorf("Expected 3 pages, but got %d", len(pages))
		}

		// Check sorting order
		if pages[0].FileName != "01.jpg" || pages[1].FileName != "02.jpeg" || pages[2].FileName != "03.png" {
			t.Errorf("Pages are not sorted correctly: got %s, %s, %s", pages[0].FileName, pages[1].FileName, pages[2].FileName)
		}

		if pages[0].Index != 0 || pages[1].Index != 1 || pages[2].Index != 2 {
			t.Errorf("Page indices are not set correctly")
		}

		if firstPageData == nil {
			t.Error("Expected first page data to be non-nil")
		}
	})

	t.Run("Unsupported type", func(t *testing.T) {
		unsupportedPath := filepath.Join(tempDir, "test.txt")
		os.WriteFile(unsupportedPath, []byte("hello"), 0644)

		_, _, err := chapterfiles.InspectChapterFile(ctx, unsupportedPath)
		if err == nil {
			t.Error("Expected an error for unsupported type, but got nil")
		}
	})
}

func TestInspectChapterFile_CBR(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	// From internal/library/chapterfiles, go up to internal/
	if err := os.Chdir("../.."); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Ensure we change back to the original directory after the test
	defer os.Chdir(originalWD)

	assetDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get asset directory: %v", err)
	}
	cbrPath := filepath.Join(assetDir, "testutil", "asset", "test.cbr")
	if _, err := os.Stat(cbrPath); err != nil {
		t.Skipf("CBR fixture not present (%s): %v", cbrPath, err)
	}
	t.Logf("Testing CBR parsing with file: %s", cbrPath)

	pages, firstPageData, err := chapterfiles.InspectChapterFile(context.Background(), cbrPath)
	if err != nil {
		t.Fatalf("InspectChapterFile failed for CBR: %v", err)
	}
	if len(pages) != 4 {
		t.Errorf("Expected 4 pages for CBR, got %d", len(pages))
	}
	if firstPageData == nil {
		t.Error("Expected first page data to be not nil")
	}
}

func TestGetChapterPage_CBZ(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	files := []struct {
		Name    string
		Content string
	}{
		{"01.jpg", "page 1 content"},
		{"02.png", "page 2 content"},
		{"03.jpeg", "page 3 content"},
	}

	cbzPath := createTestCBZWithContent(t, tempDir, "test.cbz", files)

	t.Run("Valid page index", func(t *testing.T) {
		data, filename, err := chapterfiles.GetChapterPage(ctx, cbzPath, 0)
		if err != nil {
			t.Fatalf("GetChapterPage failed: %v", err)
		}

		if filename != "01.jpg" {
			t.Errorf("Expected filename 01.jpg, got %s", filename)
		}

		if string(data) != "page 1 content" {
			t.Errorf("Expected content 'page 1 content', got %q", string(data))
		}

		// Test getting second page (index 1)
		data, filename, err = chapterfiles.GetChapterPage(ctx, cbzPath, 1)
		if err != nil {
			t.Fatalf("GetChapterPage failed for page 1: %v", err)
		}

		if filename != "02.png" {
			t.Errorf("Expected filename 02.png, got %s", filename)
		}

		if string(data) != "page 2 content" {
			t.Errorf("Expected content 'page 2 content', got %q", string(data))
		}
	})

	t.Run("Invalid page index", func(t *testing.T) {
		_, _, err := chapterfiles.GetChapterPage(ctx, cbzPath, -1)
		if err == nil {
			t.Error("Expected error for negative page index, got nil")
		}

		// Test getting page index out of bounds (10)
		_, _, err = chapterfiles.GetChapterPage(ctx, cbzPath, 10)
		if err == nil {
			t.Error("Expected error for out of bounds page index, got nil")
		}
	})

	t.Run("Unsupported type", func(t *testing.T) {
		unsupportedPath := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(unsupportedPath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create unsupported file: %v", err)
		}

		_, _, err := chapterfiles.GetChapterPage(ctx, unsupportedPath, 0)
		if err == nil {
			t.Error("Expected error for unsupported type, got nil")
		}
	})
}

func TestPageSortingConsistency(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

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

	pages, _, err := chapterfiles.InspectChapterFile(ctx, cbzPath)
	if err != nil {
		t.Fatalf("InspectChapterFile failed: %v", err)
	}

	expectedOrder := []string{"02.jpg", "003.png", "04.jpeg", "__1.jpg"}
	for i, page := range pages {
		if page.FileName != expectedOrder[i] {
			t.Errorf("Page[%d] filename = %s, want %s", i, page.FileName, expectedOrder[i])
		}
		if page.Index != i {
			t.Errorf("Page[%d] index = %d, want %d", i, page.Index, i)
		}
	}

	for i, expectedFilename := range expectedOrder {
		data, filename, err := chapterfiles.GetChapterPage(ctx, cbzPath, i)
		if err != nil {
			t.Fatalf("GetChapterPage failed for page %d: %v", i, err)
		}

		if filename != expectedFilename {
			t.Errorf("GetChapterPage(%d) returned filename %s, want %s", i, filename, expectedFilename)
		}

		if string(data) != fmt.Sprintf("page %d", i+1) {
			t.Errorf("GetChapterPage(%d) returned wrong content: %s", i, string(data))
		}
	}
}

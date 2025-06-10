// This file tests the main library scanner. It sets up a temporary
// directory structure with test archives to simulate a real library.

package library

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrsandeep/mango-go/internal/config"
)

// setupTestLibrary creates a temporary directory structure for scanner tests.
func setupTestLibrary(t *testing.T) string {
	t.Helper()

	// Create root library directory
	rootDir := t.TempDir()

	// Create Series A
	seriesADir := filepath.Join(rootDir, "Series A")
	os.Mkdir(seriesADir, 0755)
	createTestCBZ(t, seriesADir) // Will create "test.cbz" inside

	// Create Series B
	seriesBDir := filepath.Join(rootDir, "Series B")
	os.Mkdir(seriesBDir, 0755)
	// Create a fake CBR file
	cbrFile, _ := os.Create(filepath.Join(seriesBDir, "chapter2.cbr"))
	cbrFile.Close()
	// Create a non-archive file
	txtFile, _ := os.Create(filepath.Join(seriesBDir, "notes.txt"))
	txtFile.Close()

	// Create an empty directory
	os.Mkdir(filepath.Join(rootDir, "Empty Series"), 0755)

	return rootDir
}

func TestScanLibrary(t *testing.T) {
	libraryPath := setupTestLibrary(t)
	cfg := &config.Config{LibraryPath: libraryPath}

	mangaCollection, err := ScanLibrary(cfg)
	if err != nil {
		t.Fatalf("ScanLibrary failed: %v", err)
	}

	// Check number of series found (should be 2, "Empty Series" has no archives)
	if len(mangaCollection) != 2 {
		t.Errorf("Expected 2 manga series, but got %d", len(mangaCollection))
	}

	// Check Series A
	seriesA, ok := mangaCollection["Series A"]
	if !ok {
		t.Fatal("Expected to find 'Series A', but it was not found")
	}
	if len(seriesA.Chapters) != 1 {
		t.Fatalf("Expected 1 chapter in 'Series A', but got %d", len(seriesA.Chapters))
	}
	if seriesA.Chapters[0].FileName != "test.cbz" {
		t.Errorf("Unexpected chapter filename: %s", seriesA.Chapters[0].FileName)
	}
	// The test CBZ has 3 image files
	if seriesA.Chapters[0].PageCount != 3 {
		t.Errorf("Expected 3 pages in chapter, but got %d", seriesA.Chapters[0].PageCount)
	}

	// Check Series B
	seriesB, ok := mangaCollection["Series B"]
	if !ok {
		t.Fatal("Expected to find 'Series B', but it was not found")
	}
	if len(seriesB.Chapters) != 1 {
		t.Fatalf("Expected 1 chapter in 'Series B', but got %d", len(seriesB.Chapters))
	}
	// The test CBR parsing is not implemented, so page count should be 0
	if seriesB.Chapters[0].PageCount != 0 {
		t.Errorf("Expected 0 pages in CBR chapter, but got %d", seriesB.Chapters[0].PageCount)
	}
}

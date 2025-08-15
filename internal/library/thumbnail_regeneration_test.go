// This file tests the thumbnail regeneration functionality.

package library_test

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// TestRegenerateThumbnails tests the RegenerateThumbnails functionality
func TestRegenerateThumbnails(t *testing.T) {
	app := testutil.SetupTestApp(t)
	st := store.New(app.DB())
	libraryRoot := app.Config().Library.Path

	// Create test folder structure with chapters
	testFolder := filepath.Join(libraryRoot, "Test Series")
	os.MkdirAll(testFolder, 0755)

	// Create test chapters
	testutil.CreateTestCBZ(t, testFolder, "ch1.cbz", []string{"p1.jpg", "p2.jpg"})
	testutil.CreateTestCBZ(t, testFolder, "ch2.cbz", []string{"p1.jpg", "p2.jpg", "p3.jpg"})

	// Run library sync to create folders and chapters
	library.LibrarySync(&testutil.MockJobContext{App: app})

	// Verify chapters were created
	chapters, err := st.GetAllChaptersByHash()
	if err != nil {
		t.Fatalf("Failed to get chapters: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("Expected 2 chapters, got %d", len(chapters))
	}

	// Clear existing thumbnails to test regeneration
	for _, chapterInfo := range chapters {
		// Get full chapter to access thumbnail
		chapter, err := st.GetChapterByID(chapterInfo.ID, 1) // Use user ID 1 for testing
		if err != nil {
			t.Fatalf("Failed to get chapter: %v", err)
		}
		err = st.UpdateChapterThumbnail(chapter.ID, "")
		if err != nil {
			t.Fatalf("Failed to clear thumbnail: %v", err)
		}
	}

	// Run thumbnail regeneration
	ctx := &testutil.MockJobContext{App: app}
	library.RegenerateThumbnails(ctx)

	// Verify thumbnails were regenerated
	chapters, err = st.GetAllChaptersByHash()
	if err != nil {
		t.Fatalf("Failed to get chapters after regeneration: %v", err)
	}

	for _, chapterInfo := range chapters {
		// Get full chapter to access thumbnail
		chapter, err := st.GetChapterByID(chapterInfo.ID, 1) // Use user ID 1 for testing
		if err != nil {
			t.Fatalf("Failed to get chapter: %v", err)
		}
		if chapter.Thumbnail == "" {
			t.Errorf("Chapter %s should have a thumbnail after regeneration", chapter.Path)
		}
	}

	// Verify folder thumbnails were also updated
	folders, err := st.GetAllFoldersByPath()
	if err != nil {
		t.Fatalf("Failed to get folders: %v", err)
	}

	testSeriesPath := filepath.Join(libraryRoot, "Test Series")
	if folder, exists := folders[testSeriesPath]; exists {
		if folder.Thumbnail == "" {
			t.Error("Folder should have a thumbnail after regeneration")
		}
	} else {
		t.Error("Test series folder should exist")
	}
}

// TestRegenerateThumbnailsWithNoChapters tests the case where there are no chapters to process
func TestRegenerateThumbnailsWithNoChapters(t *testing.T) {
	app := testutil.SetupTestApp(t)
	ctx := &testutil.MockJobContext{App: app}

	// Run thumbnail regeneration on empty library
	library.RegenerateThumbnails(ctx)

	// Should complete without error
	// This test mainly ensures the function doesn't panic on empty data
}

// TestRegenerateThumbnailsWithCorruptedChapters tests handling of corrupted chapters during regeneration
func TestRegenerateThumbnailsWithCorruptedChapters(t *testing.T) {
	app := testutil.SetupTestApp(t)
	st := store.New(app.DB())
	libraryRoot := app.Config().Library.Path

	// Create test folder
	testFolder := filepath.Join(libraryRoot, "Corrupted Series")
	os.MkdirAll(testFolder, 0755)

	// Create a valid chapter first
	testutil.CreateTestCBZ(t, testFolder, "valid.cbz", []string{"p1.jpg"})

	// Create a corrupted chapter
	corruptedChapterPath := filepath.Join(testFolder, "corrupted.cbz")
	err := os.WriteFile(corruptedChapterPath, []byte("This is not a valid CBZ"), 0644)
	if err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}

	// Run library sync to create the structure
	library.LibrarySync(&testutil.MockJobContext{App: app})

	// Verify chapters were created (including the corrupted one)
	chapters, err := st.GetAllChaptersByHash()
	if err != nil {
		t.Fatalf("Failed to get chapters: %v", err)
	}

	// Clear existing thumbnails
	for _, chapterInfo := range chapters {
		// Get full chapter to access thumbnail
		chapter, err := st.GetChapterByID(chapterInfo.ID, 1) // Use user ID 1 for testing
		if err != nil {
			t.Fatalf("Failed to get chapter: %v", err)
		}
		err = st.UpdateChapterThumbnail(chapter.ID, "")
		if err != nil {
			t.Fatalf("Failed to clear thumbnail: %v", err)
		}
	}

	// Run thumbnail regeneration
	ctx := &testutil.MockJobContext{App: app}
	library.RegenerateThumbnails(ctx)

	// Verify that valid chapters got thumbnails
	chapters, err = st.GetAllChaptersByHash()
	if err != nil {
		t.Fatalf("Failed to get chapters after regeneration: %v", err)
	}

	var validChapterFound bool
	for _, chapterInfo := range chapters {
		if filepath.Base(chapterInfo.Path) == "valid.cbz" {
			// Get full chapter to access thumbnail
			chapter, err := st.GetChapterByID(chapterInfo.ID, 1) // Use user ID 1 for testing
			if err != nil {
				t.Fatalf("Failed to get chapter: %v", err)
			}
			validChapterFound = true
			if chapter.Thumbnail == "" {
				t.Error("Valid chapter should have a thumbnail after regeneration")
			}
		}
	}

	if !validChapterFound {
		t.Error("Valid chapter should exist after regeneration")
	}
}

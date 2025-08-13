// This file tests the main library scanner.

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

// Run the whole test suite as it is a sequential test.
func TestLibrarySync(t *testing.T) {
	app := testutil.SetupTestApp(t) // Sets up in-memory DB, config, etc.
	st := store.New(app.DB())
	libraryRoot := app.Config().Library.Path

	// --- Test 1: Initial Scan ---
	t.Run("Initial Scan", func(t *testing.T) {
		// Create mock file structure
		os.MkdirAll(filepath.Join(libraryRoot, "Series A", "Volume 1"), 0755)
		testutil.CreateTestCBZ(t, filepath.Join(libraryRoot, "Series A", "Volume 1"), "ch1.cbz", []string{"p1.jpg"})

		library.LibrarySync(app)

		// Verify folder structure
		folders, _ := st.GetAllFoldersByPath()
		if len(folders) != 2 {
			t.Fatalf("Expected 2 folders, got %d", len(folders))
		}
		if _, ok := folders[filepath.Join(libraryRoot, "Series A")]; !ok {
			t.Error("Series A folder not created")
		}
		if _, ok := folders[filepath.Join(libraryRoot, "Series A", "Volume 1")]; !ok {
			t.Error("Volume 1 folder not created")
		}

		// Verify chapter
		chapters, _ := st.GetAllChaptersByHash()
		if len(chapters) != 1 {
			t.Fatalf("Expected 1 chapter, got %d", len(chapters))
		}
		keys := make([]string, 0, len(chapters))
		for k := range chapters {
			keys = append(keys, k)
		}
		if len(keys) != 1 {
			t.Fatalf("Expected 1 chapter, got %d", len(keys))
		}
		if chapters[keys[0]].Path != filepath.Join(libraryRoot, "Series A", "Volume 1", "ch1.cbz") {
			t.Errorf("Expected chapter path '%s', got '%s'", filepath.Join(libraryRoot, "Series A", "Volume 1", "ch1.cbz"), chapters[keys[0]].Path)
		}

		// Check if the chapter was created correctly
		var pageCount int
		var chapterPath string
		err := app.DB().QueryRow("SELECT path, page_count FROM chapters WHERE id = ?", chapters[keys[0]].ID).Scan(&chapterPath, &pageCount)
		if err != nil {
			t.Fatalf("Failed to find chapter for 'Series A': %v", err)
		}
		if pageCount != 1 {
			t.Errorf("Expected page count of 1 for scanned chapter, got %d", pageCount)
		}
		expectedPath := filepath.Join(libraryRoot, "Series A", "Volume 1", "ch1.cbz")
		if chapterPath != expectedPath {
			t.Errorf("Expected chapter path '%s', got '%s'", expectedPath, chapterPath)
		}
	})

	// --- Test 2: Moved File ---
	t.Run("Moved File Detection", func(t *testing.T) {
		// Move the file
		os.MkdirAll(filepath.Join(libraryRoot, "Series B"), 0755)
		oldPath := filepath.Join(libraryRoot, "Series A", "Volume 1", "ch1.cbz")
		newPath := filepath.Join(libraryRoot, "Series B", "ch1-moved.cbz")
		err := os.Rename(oldPath, newPath)
		if err != nil {
			t.Fatalf("Failed to move file: %v", err)
		}

		library.LibrarySync(app)

		chapters, _ := st.GetAllChaptersByHash()
		if len(chapters) != 1 {
			t.Fatalf("Expected 1 chapter after move, got %d", len(chapters))
		}

		var found bool
		for _, ch := range chapters {
			if ch.Path == newPath {
				found = true
			}
		}
		if !found {
			t.Error("Chapter path was not updated after move")
		}

		folders, _ := st.GetAllFoldersByPath()
		// After moving the file, only Series B should remain (Series A and Volume 1 become empty)
		if len(folders) != 1 {
			t.Fatalf("Expected 1 folder after move (empty folders pruned), got %d", len(folders))
		}
		if _, ok := folders[filepath.Join(libraryRoot, "Series B")]; !ok {
			t.Error("Series B folder not created")
		}
		// Series A and Volume 1 should be pruned since they're now empty
		if _, ok := folders[filepath.Join(libraryRoot, "Series A")]; ok {
			t.Error("Series A folder should have been pruned since it's empty")
		}
		if _, ok := folders[filepath.Join(libraryRoot, "Series A", "Volume 1")]; ok {
			t.Error("Volume 1 folder should have been pruned since it's empty")
		}
	})

	// --- Test 3: Pruning ---
	t.Run("Pruning Deleted File", func(t *testing.T) {
		err := os.Remove(filepath.Join(libraryRoot, "Series B", "ch1-moved.cbz"))
		if err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		library.LibrarySync(app)

		chapters, _ := st.GetAllChaptersByHash()
		if len(chapters) != 0 {
			t.Errorf("Expected 0 chapters after pruning, got %d", len(chapters))
		}

		folders, _ := st.GetAllFoldersByPath()
		// After deleting the last chapter, Series B should also be pruned
		if len(folders) != 0 {
			t.Fatalf("Expected 0 folders after pruning all empty folders, got %d", len(folders))
		}
		// Series B folder should be pruned since it's now empty
		if _, ok := folders[filepath.Join(libraryRoot, "Series B")]; ok {
			t.Error("Series B folder should have been pruned since it's empty")
		}
	})

	// --- Test 4: Empty Directory Ignored ---
	t.Run("Empty Directory Ignored", func(t *testing.T) {
		// Create an empty directory
		emptyDir := filepath.Join(libraryRoot, "Empty Series")
		os.MkdirAll(emptyDir, 0755)

		library.LibrarySync(app)

		folders, _ := st.GetAllFoldersByPath()
		// Should still have 0 folders since all previous folders were empty
		if len(folders) != 0 {
			t.Fatalf("Expected 0 folders, empty directory should be ignored, got %d", len(folders))
		}
		if _, ok := folders[emptyDir]; ok {
			t.Error("Empty directory should not be created in database")
		}
	})

	// --- Test 5: Nested Folder Parent ID Linking ---
	t.Run("Nested Folder Parent ID Linking", func(t *testing.T) {
		// Create a nested folder structure with manga archives
		nestedPath := filepath.Join(libraryRoot, "Nested Series", "Volume 1", "Chapter 1")
		os.MkdirAll(nestedPath, 0755)
		testutil.CreateTestCBZ(t, nestedPath, "ch1.cbz", []string{"p1.jpg"})

		library.LibrarySync(app)

		folders, _ := st.GetAllFoldersByPath()
		// Should have 3 folders: Nested Series, Volume 1, and Chapter 1
		if len(folders) != 3 {
			t.Fatalf("Expected 3 folders, got %d", len(folders))
		}

		// Check that all folders exist
		nestedSeriesPath := filepath.Join(libraryRoot, "Nested Series")
		volume1Path := filepath.Join(libraryRoot, "Nested Series", "Volume 1")
		chapter1Path := filepath.Join(libraryRoot, "Nested Series", "Volume 1", "Chapter 1")

		if _, ok := folders[nestedSeriesPath]; !ok {
			t.Error("Nested Series folder not created")
		}
		if _, ok := folders[volume1Path]; !ok {
			t.Error("Volume 1 folder not created")
		}
		if _, ok := folders[chapter1Path]; !ok {
			t.Error("Chapter 1 folder not created")
		}

		// Verify parent-child relationships
		nestedSeries := folders[nestedSeriesPath]
		volume1 := folders[volume1Path]
		chapter1 := folders[chapter1Path]

		// Nested Series should have no parent (root level)
		if nestedSeries.ParentID != nil {
			t.Errorf("Nested Series should have no parent, got ParentID: %v", nestedSeries.ParentID)
		}

		// Volume 1 should have Nested Series as parent
		if volume1.ParentID == nil {
			t.Error("Volume 1 should have Nested Series as parent")
		} else if *volume1.ParentID != nestedSeries.ID {
			t.Errorf("Volume 1 ParentID should be %d, got %d", nestedSeries.ID, *volume1.ParentID)
		}

		// Chapter 1 should have Volume 1 as parent
		if chapter1.ParentID == nil {
			t.Error("Chapter 1 should have Volume 1 as parent")
		} else if *chapter1.ParentID != volume1.ID {
			t.Errorf("Chapter 1 ParentID should be %d, got %d", volume1.ID, *chapter1.ParentID)
		}

		// Verify chapter was created
		chapters, _ := st.GetAllChaptersByHash()
		if len(chapters) != 1 {
			t.Fatalf("Expected 1 chapter, got %d", len(chapters))
		}
	})
}

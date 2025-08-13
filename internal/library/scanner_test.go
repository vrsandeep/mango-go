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

	// --- Test 6: Bad Files Detection During Sync ---
	t.Run("Bad Files Detection During Sync", func(t *testing.T) {
		// Create a corrupted/invalid archive file
		badFilePath := filepath.Join(libraryRoot, "Bad Series", "bad-chapter.cbz")
		os.MkdirAll(filepath.Dir(badFilePath), 0755)

		// Create an invalid file (not a real CBZ)
		err := os.WriteFile(badFilePath, []byte("This is not a valid CBZ file"), 0644)
		if err != nil {
			t.Fatalf("Failed to create bad file: %v", err)
		}

		library.LibrarySync(app)

		// Check that the bad file was recorded
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		if len(badFiles) == 0 {
			t.Error("Expected bad files to be detected during sync")
		}

		// Find our specific bad file
		var foundBadFile bool
		for _, badFile := range badFiles {
			if badFile.Path == badFilePath {
				foundBadFile = true
				if badFile.Error == "" {
					t.Error("Bad file should have an error message")
				}
				break
			}
		}
		if !foundBadFile {
			t.Error("Bad file was not recorded in database")
		}

		// Verify that the bad file directory was created (since folders are processed before chapters)
		// but the bad file itself was recorded
		folders, _ := st.GetAllFoldersByPath()
		if _, ok := folders[filepath.Dir(badFilePath)]; !ok {
			t.Error("Directory should be created even if it contains bad files")
		}
	})

	// --- Test 7: Bad Files Cleanup After Fix ---
	t.Run("Bad Files Cleanup After Fix", func(t *testing.T) {
		// Get the bad file we created in previous test
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		if len(badFiles) == 0 {
			t.Fatal("No bad files found for cleanup test")
		}

		// Fix the bad file by replacing it with a valid CBZ
		badFilePath := filepath.Join(libraryRoot, "Bad Series", "bad-chapter.cbz")
		os.Remove(badFilePath) // Remove the bad file
		testutil.CreateTestCBZ(t, filepath.Dir(badFilePath), "bad-chapter.cbz", []string{"p1.jpg"})

		library.LibrarySync(app)

		// Check that the bad file record was removed
		badFilesAfter, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files after fix: %v", err)
		}

		// The bad file should be removed from the list
		var stillBad bool
		for _, badFile := range badFilesAfter {
			if badFile.Path == badFilePath {
				stillBad = true
				break
			}
		}
		if stillBad {
			t.Error("Bad file record should have been removed after fixing the file")
		}

		// Verify that the directory is now properly created
		folders, _ := st.GetAllFoldersByPath()
		badSeriesPath := filepath.Join(libraryRoot, "Bad Series")
		if _, ok := folders[badSeriesPath]; !ok {
			t.Error("Bad Series folder should now be created after fixing the file")
		}
	})

	// --- Test 8: Bad Files Cleanup for Deleted Files ---
	t.Run("Bad Files Cleanup for Deleted Files", func(t *testing.T) {
		// Create a bad file
		badFilePath := filepath.Join(libraryRoot, "Temp Bad Series", "temp-bad.cbz")
		os.MkdirAll(filepath.Dir(badFilePath), 0755)
		err := os.WriteFile(badFilePath, []byte("Invalid file"), 0644)
		if err != nil {
			t.Fatalf("Failed to create temporary bad file: %v", err)
		}

		// Run sync to record the bad file
		library.LibrarySync(app)

		// Verify bad file was recorded
		badFileStore := store.NewBadFileStore(app.DB())
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		var tempBadFileCount int
		for _, badFile := range badFiles {
			if filepath.Dir(badFile.Path) == filepath.Dir(badFilePath) {
				tempBadFileCount++
			}
		}
		if tempBadFileCount == 0 {
			t.Fatal("Temporary bad file was not recorded")
		}

		// Delete the bad file and its directory
		os.RemoveAll(filepath.Dir(badFilePath))

		// Run sync again to trigger cleanup
		library.LibrarySync(app)

		// Check that bad file records for deleted files were cleaned up
		badFilesAfter, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files after cleanup: %v", err)
		}

		// Count remaining bad files in the deleted directory
		var remainingBadFiles int
		for _, badFile := range badFilesAfter {
			if filepath.Dir(badFile.Path) == filepath.Dir(badFilePath) {
				remainingBadFiles++
			}
		}
		if remainingBadFiles > 0 {
			t.Errorf("Expected 0 bad files after cleanup, got %d", remainingBadFiles)
		}
	})
}

// Test for bad files handling
func TestBadFiles(t *testing.T) {
	app := testutil.SetupTestApp(t)
	badFileStore := store.NewBadFileStore(app.DB())
	libraryRoot := app.Config().Library.Path

	t.Run("Test checkBadFilesDuringSync", func(t *testing.T) {
		// Create a test directory with both good and bad files
		testDir := filepath.Join(libraryRoot, "Test Bad Files")
		os.MkdirAll(testDir, 0755)

		// Create a good CBZ file
		goodFile := filepath.Join(testDir, "good.cbz")
		testutil.CreateTestCBZ(t, testDir, "good.cbz", []string{"p1.jpg"})

		// Create a bad file
		badFile := filepath.Join(testDir, "bad.cbz")
		err := os.WriteFile(badFile, []byte("Invalid archive"), 0644)
		if err != nil {
			t.Fatalf("Failed to create bad file: %v", err)
		}

		// Test the functionality through LibrarySync
		library.LibrarySync(app)

		// Verify results
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		// Should have at least one bad file
		if len(badFiles) == 0 {
			t.Error("Expected bad files to be detected")
		}

		// Check that the bad file was recorded
		var foundBadFile bool
		for _, bf := range badFiles {
			if bf.Path == badFile {
				foundBadFile = true
				break
			}
		}
		if !foundBadFile {
			t.Error("Bad file was not recorded")
		}

		// Check that the good file was not recorded as bad
		var foundGoodFile bool
		for _, bf := range badFiles {
			if bf.Path == goodFile {
				foundGoodFile = true
				break
			}
		}
		if foundGoodFile {
			t.Error("Good file should not be recorded as bad")
		}
	})

	t.Run("Test cleanupMissingBadFileRecords", func(t *testing.T) {
		// Create a bad file record for a file that doesn't exist
		nonExistentPath := filepath.Join(libraryRoot, "Non Existent", "missing.cbz")
		err := badFileStore.CreateBadFile(nonExistentPath, "test_error", 0)
		if err != nil {
			t.Fatalf("Failed to create bad file record: %v", err)
		}

		// Verify it was created
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		var foundNonExistent bool
		for _, bf := range badFiles {
			if bf.Path == nonExistentPath {
				foundNonExistent = true
				break
			}
		}
		if !foundNonExistent {
			t.Fatal("Non-existent bad file record was not created")
		}

		// Test cleanup through LibrarySync
		library.LibrarySync(app)

		// Check that the bad file record was cleaned up
		badFilesAfter, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files after cleanup: %v", err)
		}

		// The non-existent file record should be removed
		var stillExists bool
		for _, bf := range badFilesAfter {
			if bf.Path == nonExistentPath {
				stillExists = true
				break
			}
		}
		if stillExists {
			t.Error("Bad file record for non-existent file should have been cleaned up")
		}
	})

	t.Run("Test hasMangaArchives", func(t *testing.T) {
		// Test directory with manga archives
		hasArchivesDir := filepath.Join(libraryRoot, "Has Archives")
		os.MkdirAll(hasArchivesDir, 0755)
		testutil.CreateTestCBZ(t, hasArchivesDir, "test.cbz", []string{"p1.jpg"})

		// Test empty directory
		emptyDir := filepath.Join(libraryRoot, "Empty Dir")
		os.MkdirAll(emptyDir, 0755)

		// Test directory with non-archive files
		nonArchiveDir := filepath.Join(libraryRoot, "Non Archive")
		os.MkdirAll(nonArchiveDir, 0755)
		err := os.WriteFile(filepath.Join(nonArchiveDir, "readme.txt"), []byte("Not an archive"), 0644)
		if err != nil {
			t.Fatalf("Failed to create text file: %v", err)
		}

		// Test hasMangaArchives function (we need to make it accessible for testing)
		// For now, we'll test it indirectly through LibrarySync
		library.LibrarySync(app)

		folders, _ := store.New(app.DB()).GetAllFoldersByPath()

		// Directory with archives should be created
		if _, ok := folders[hasArchivesDir]; !ok {
			t.Error("Directory with manga archives should be created")
		}

		// Empty directory should not be created
		if _, ok := folders[emptyDir]; ok {
			t.Error("Empty directory should not be created")
		}

		// Directory with non-archives should not be created
		if _, ok := folders[nonArchiveDir]; ok {
			t.Error("Directory with non-archive files should not be created")
		}
	})
}

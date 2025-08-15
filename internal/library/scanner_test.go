// This file tests the main library scanner.

package library_test

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// Helper functions to reduce code duplication
func assertFolderCount(t *testing.T, st *store.Store, expected int, msg string) {
	folders, _ := st.GetAllFoldersByPath()
	if len(folders) != expected {
		t.Fatalf("%s: expected %d folders, got %d", msg, expected, len(folders))
	}
}

func assertChapterCount(t *testing.T, st *store.Store, expected int, msg string) {
	chapters, _ := st.GetAllChaptersByHash()
	if len(chapters) != expected {
		t.Fatalf("%s: expected %d chapters, got %d", msg, expected, len(chapters))
	}
}

func assertFolderExists(t *testing.T, folders map[string]*models.Folder, path string, shouldExist bool, msg string) {
	_, exists := folders[path]
	if shouldExist && !exists {
		t.Errorf("%s: folder '%s' should exist", msg, path)
	} else if !shouldExist && exists {
		t.Errorf("%s: folder '%s' should not exist", msg, path)
	}
}

func assertBadFileRecorded(t *testing.T, badFileStore *store.BadFileStore, filePath string, shouldExist bool, msg string) {
	badFiles, err := badFileStore.GetAllBadFiles()
	if err != nil {
		t.Fatalf("Failed to get bad files: %v", err)
	}

	var found bool
	for _, bf := range badFiles {
		if bf.Path == filePath {
			found = true
			break
		}
	}

	if shouldExist && !found {
		t.Errorf("%s: bad file '%s' should be recorded", msg, filePath)
	} else if !shouldExist && found {
		t.Errorf("%s: bad file '%s' should not be recorded", msg, filePath)
	}
}

func assertParentChildRelationship(t *testing.T, parent, child *models.Folder, parentName, childName string) {
	if child.ParentID == nil {
		t.Errorf("%s should have %s as parent", childName, parentName)
	} else if *child.ParentID != parent.ID {
		t.Errorf("%s ParentID should be %d, got %d", childName, parent.ID, *child.ParentID)
	}
}

func assertBadFileCountInDirectory(t *testing.T, badFileStore *store.BadFileStore, dirPath string, expectedCount int, msg string) {
	badFiles, err := badFileStore.GetAllBadFiles()
	if err != nil {
		t.Fatalf("Failed to get bad files: %v", err)
	}

	var count int
	for _, badFile := range badFiles {
		if filepath.Dir(badFile.Path) == dirPath {
			count++
		}
	}

	if count != expectedCount {
		t.Errorf("%s: expected %d bad files in directory '%s', got %d", msg, expectedCount, dirPath, count)
	}
}

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
		assertFolderCount(t, st, 2, "Initial scan")
		assertFolderExists(t, folders, filepath.Join(libraryRoot, "Series A"), true, "Initial scan")
		assertFolderExists(t, folders, filepath.Join(libraryRoot, "Series A", "Volume 1"), true, "Initial scan")

		// Verify chapter
		chapters, _ := st.GetAllChaptersByHash()
		assertChapterCount(t, st, 1, "Initial scan")

		// Check chapter details
		var pageCount int
		var chapterPath string
		keys := make([]string, 0, len(chapters))
		for k := range chapters {
			keys = append(keys, k)
		}
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

		assertChapterCount(t, st, 1, "After file move")
		assertFolderCount(t, st, 1, "After file move")

		// Verify chapter was moved
		chapters, _ := st.GetAllChaptersByHash()
		assertChapterCount(t, st, 1, "After file move")

		var found bool
		for _, ch := range chapters {
			if ch.Path == newPath {
				found = true
				break
			}
		}
		if !found {
			t.Error("Chapter path was not updated after move")
		}

		// Check folder pruning
		folders, _ := st.GetAllFoldersByPath()
		assertFolderCount(t, st, 1, "After file move")
		assertFolderExists(t, folders, filepath.Join(libraryRoot, "Series B"), true, "After file move")
		assertFolderExists(t, folders, filepath.Join(libraryRoot, "Series A"), false, "After file move")
		assertFolderExists(t, folders, filepath.Join(libraryRoot, "Series A", "Volume 1"), false, "After file move")
	})

	// --- Test 3: Pruning ---
	t.Run("Pruning Deleted File", func(t *testing.T) {
		err := os.Remove(filepath.Join(libraryRoot, "Series B", "ch1-moved.cbz"))
		if err != nil {
			t.Fatalf("Failed to delete file: %v", err)
		}

		library.LibrarySync(app)

		assertChapterCount(t, st, 0, "After pruning")
		// After deleting the last chapter, Series B should also be pruned
		assertFolderCount(t, st, 0, "After pruning")
	})

	// --- Test 4: Empty Directory Ignored ---
	t.Run("Empty Directory Ignored", func(t *testing.T) {
		// Create an empty directory
		emptyDir := filepath.Join(libraryRoot, "Empty Series")
		os.MkdirAll(emptyDir, 0755)

		library.LibrarySync(app)

		assertFolderCount(t, st, 0, "Empty directory test")
	})

	// --- Test 5: Nested Folder Parent ID Linking ---
	t.Run("Nested Folder Parent ID Linking", func(t *testing.T) {
		// Create a nested folder structure with manga archives
		nestedPath := filepath.Join(libraryRoot, "Nested Series", "Volume 1", "Chapter 1")
		os.MkdirAll(nestedPath, 0755)
		testutil.CreateTestCBZ(t, nestedPath, "ch1.cbz", []string{"p1.jpg"})

		library.LibrarySync(app)

		folders, _ := st.GetAllFoldersByPath()
		assertFolderCount(t, st, 3, "Nested folder test")

		// Check that all folders exist
		nestedSeriesPath := filepath.Join(libraryRoot, "Nested Series")
		volume1Path := filepath.Join(libraryRoot, "Nested Series", "Volume 1")
		chapter1Path := filepath.Join(libraryRoot, "Nested Series", "Volume 1", "Chapter 1")

		assertFolderExists(t, folders, nestedSeriesPath, true, "Nested folder test")
		assertFolderExists(t, folders, volume1Path, true, "Nested folder test")
		assertFolderExists(t, folders, chapter1Path, true, "Nested folder test")

		// Verify parent-child relationships
		nestedSeries := folders[nestedSeriesPath]
		volume1 := folders[volume1Path]
		chapter1 := folders[chapter1Path]

		// Nested Series should have no parent (root level)
		if nestedSeries.ParentID != nil {
			t.Errorf("Nested Series should have no parent, got ParentID: %v", nestedSeries.ParentID)
		}

		// Check parent-child relationships
		assertParentChildRelationship(t, nestedSeries, volume1, "Nested Series", "Volume 1")
		assertParentChildRelationship(t, volume1, chapter1, "Volume 1", "Chapter 1")

		// Verify chapter was created
		assertChapterCount(t, st, 1, "Nested folder test")
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
		assertBadFileRecorded(t, badFileStore, badFilePath, true, "Bad file detection")

		// Verify that the bad file directory was created (since folders are processed before chapters)
		// but the bad file itself was recorded
		folders, _ := st.GetAllFoldersByPath()
		assertFolderExists(t, folders, filepath.Dir(badFilePath), true, "Bad file directory creation")
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
		assertBadFileRecorded(t, badFileStore, badFilePath, false, "Bad file cleanup")

		// Verify that the directory is now properly created
		folders, _ := st.GetAllFoldersByPath()
		badSeriesPath := filepath.Join(libraryRoot, "Bad Series")
		assertFolderExists(t, folders, badSeriesPath, true, "Folder is not bad anymore. It should be created")
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
		assertBadFileCountInDirectory(t, badFileStore, filepath.Dir(badFilePath), 1, "Temporary bad file recording")

		// Delete the bad file and its directory
		os.RemoveAll(filepath.Dir(badFilePath))

		// Run sync again to trigger cleanup
		library.LibrarySync(app)

		// Check that bad file records for deleted files were cleaned up
		assertBadFileCountInDirectory(t, badFileStore, filepath.Dir(badFilePath), 0, "Bad file cleanup")
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
		assertBadFileRecorded(t, badFileStore, badFile, true, "Bad file detection")
		assertBadFileRecorded(t, badFileStore, goodFile, false, "Good file protection")
	})

	t.Run("Test cleanupMissingBadFileRecords", func(t *testing.T) {
		// Create a bad file record for a file that doesn't exist
		nonExistentPath := filepath.Join(libraryRoot, "Non Existent", "missing.cbz")
		err := badFileStore.CreateBadFile(nonExistentPath, "test_error", 0)
		if err != nil {
			t.Fatalf("Failed to create bad file record: %v", err)
		}

		// Verify it was created
		assertBadFileRecorded(t, badFileStore, nonExistentPath, true, "Bad file creation")

		// Test cleanup through LibrarySync
		library.LibrarySync(app)

		// Check that the bad file record was cleaned up
		assertBadFileRecorded(t, badFileStore, nonExistentPath, false, "Bad file cleanup")
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
		assertFolderExists(t, folders, hasArchivesDir, true, "Directory with archives should be created")
		// Empty directory should not be created
		assertFolderExists(t, folders, emptyDir, false, "Empty directory should not be created")
		// Directory with non-archives should not be created
		assertFolderExists(t, folders, nonArchiveDir, false, "Directory with non-archive files should not be created")
	})
}

// TestCorruptedChaptersPruning tests the enhanced prune function's ability to remove corrupted chapters
func TestCorruptedChaptersPruning(t *testing.T) {
	app := testutil.SetupTestApp(t)
	st := store.New(app.DB())
	libraryRoot := app.Config().Library.Path

	// Clean up any existing data from previous tests by removing all files
	os.RemoveAll(libraryRoot)
	os.MkdirAll(libraryRoot, 0755)

	// Create a valid chapter first
	validChapterPath := filepath.Join(libraryRoot, "Corruption Test", "valid-chapter.cbz")
	os.MkdirAll(filepath.Dir(validChapterPath), 0755)
	testutil.CreateTestCBZ(t, filepath.Dir(validChapterPath), "valid-chapter.cbz", []string{"p1.jpg"})

	// Run sync to create the chapter in the database
	library.LibrarySync(app)

	// Verify chapter was created
	assertChapterCount(t, st, 1, "After creating valid chapter")

	// Corrupt the chapter file
	err := os.WriteFile(validChapterPath, []byte("This is now a corrupted file"), 0644)
	if err != nil {
		t.Fatalf("Failed to corrupt chapter file: %v", err)
	}

	// Run sync again - the corrupted chapter should be pruned
	library.LibrarySync(app)

	// Verify corrupted chapter was pruned
	assertChapterCount(t, st, 0, "After pruning corrupted chapter")

	// Verify the folder still exists (since it was created before corruption)
	folders, _ := st.GetAllFoldersByPath()
	corruptionTestPath := filepath.Join(libraryRoot, "Corruption Test")
	assertFolderExists(t, folders, corruptionTestPath, true, "Corruption test folder should still exist")

	// Verify bad file was recorded
	badFileStore := store.NewBadFileStore(app.DB())
	assertBadFileRecorded(t, badFileStore, validChapterPath, true, "Corrupted chapter should be recorded as bad file")
}

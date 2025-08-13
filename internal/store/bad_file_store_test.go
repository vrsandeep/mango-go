package store_test

import (
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

func TestBadFileStore(t *testing.T) {
	app := testutil.SetupTestApp(t)
	badFileStore := store.NewBadFileStore(app.DB())

	t.Run("CreateBadFile", func(t *testing.T) {
		// Test creating a new bad file
		path := "/test/path/bad-file.cbz"
		errorType := "corrupted_archive"
		fileSize := int64(1024)

		err := badFileStore.CreateBadFile(path, errorType, fileSize)
		if err != nil {
			t.Fatalf("Failed to create bad file: %v", err)
		}

		// Verify the bad file was created
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		if len(badFiles) == 0 {
			t.Fatal("Expected bad file to be created")
		}

		createdFile := badFiles[0]
		if createdFile.Path != path {
			t.Errorf("Expected path %s, got %s", path, createdFile.Path)
		}
		if createdFile.FileName != "bad-file.cbz" {
			t.Errorf("Expected filename 'bad-file.cbz', got %s", createdFile.FileName)
		}
		if createdFile.Error != errorType {
			t.Errorf("Expected error %s, got %s", errorType, createdFile.Error)
		}
		if createdFile.FileSize != fileSize {
			t.Errorf("Expected file size %d, got %d", fileSize, createdFile.FileSize)
		}
		if createdFile.DetectedAt.IsZero() {
			t.Error("Expected DetectedAt to be set")
		}
		if createdFile.LastChecked.IsZero() {
			t.Error("Expected LastChecked to be set")
		}
	})

	t.Run("CreateBadFile_ReplaceExisting", func(t *testing.T) {
		// Test that creating a bad file with the same path replaces the existing one
		path := "/test/path/replace-test.cbz"
		initialError := "corrupted_archive"
		updatedError := "invalid_format"
		fileSize := int64(2048)

		// Create initial bad file
		err := badFileStore.CreateBadFile(path, initialError, fileSize)
		if err != nil {
			t.Fatalf("Failed to create initial bad file: %v", err)
		}

		// Get the initial file to verify it was created
		initialFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get initial bad files: %v", err)
		}

		var initialFound bool
		for _, f := range initialFiles {
			if f.Path == path {
				initialFound = true
				break
			}
		}
		if !initialFound {
			t.Fatal("Initial bad file was not created")
		}

		// Create bad file with same path but different error
		err = badFileStore.CreateBadFile(path, updatedError, fileSize)
		if err != nil {
			t.Fatalf("Failed to replace bad file: %v", err)
		}

		// Verify the file was replaced
		updatedFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get updated bad files: %v", err)
		}

		var found bool
		for _, f := range updatedFiles {
			if f.Path == path {
				found = true
				if f.Error != updatedError {
					t.Errorf("Expected error %s, got %s", updatedError, f.Error)
				}
				// Note: SQLite INSERT OR REPLACE may generate a new ID
				// The important thing is that the path and error are updated
				break
			}
		}
		if !found {
			t.Error("Expected bad file to be replaced")
		}
	})

	t.Run("GetAllBadFiles_Empty", func(t *testing.T) {
		// Test getting all bad files when none exist
		// First, clear all bad files
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		for _, bf := range badFiles {
			err := badFileStore.DeleteBadFile(bf.ID)
			if err != nil {
				t.Fatalf("Failed to delete bad file: %v", err)
			}
		}

		// Now get all bad files
		emptyFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get empty bad files: %v", err)
		}

		if len(emptyFiles) != 0 {
			t.Errorf("Expected 0 bad files, got %d", len(emptyFiles))
		}

		// Ensure it returns an empty slice, not nil
		if emptyFiles == nil {
			t.Error("Expected empty slice, got nil")
		}
	})

	t.Run("GetAllBadFiles_Ordering", func(t *testing.T) {
		// Test that bad files are returned in correct order (detected_at DESC)
		// Create multiple bad files with different timestamps
		paths := []string{
			"/test/path/first.cbz",
			"/test/path/second.cbz",
			"/test/path/third.cbz",
		}

		for _, path := range paths {
			err := badFileStore.CreateBadFile(path, "test_error", 1024)
			if err != nil {
				t.Fatalf("Failed to create bad file %s: %v", path, err)
			}
			// Small delay to ensure different timestamps
			time.Sleep(10 * time.Millisecond)
		}

		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		if len(badFiles) < 3 {
			t.Fatalf("Expected at least 3 bad files, got %d", len(badFiles))
		}

		// Check that files are ordered by detected_at DESC (newest first)
		for i := 0; i < len(badFiles)-1; i++ {
			if badFiles[i].DetectedAt.Before(badFiles[i+1].DetectedAt) {
				t.Errorf("Bad files not ordered correctly: %s should come before %s",
					badFiles[i+1].Path, badFiles[i].Path)
			}
		}
	})

	t.Run("DeleteBadFile_ByID", func(t *testing.T) {
		// Test deleting a bad file by ID
		path := "/test/path/delete-by-id.cbz"
		err := badFileStore.CreateBadFile(path, "test_error", 1024)
		if err != nil {
			t.Fatalf("Failed to create bad file: %v", err)
		}

		// Get the file to get its ID
		badFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get bad files: %v", err)
		}

		var fileID int64
		for _, f := range badFiles {
			if f.Path == path {
				fileID = f.ID
				break
			}
		}

		if fileID == 0 {
			t.Fatal("Failed to find created bad file")
		}

		// Delete the file
		err = badFileStore.DeleteBadFile(fileID)
		if err != nil {
			t.Fatalf("Failed to delete bad file: %v", err)
		}

		// Verify it was deleted
		remainingFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get remaining bad files: %v", err)
		}

		for _, f := range remainingFiles {
			if f.ID == fileID {
				t.Error("Bad file was not deleted")
			}
		}
	})

	t.Run("DeleteBadFile_ByPath", func(t *testing.T) {
		// Test deleting a bad file by path
		path := "/test/path/delete-by-path.cbz"
		err := badFileStore.CreateBadFile(path, "test_error", 1024)
		if err != nil {
			t.Fatalf("Failed to create bad file: %v", err)
		}

		// Delete the file by path
		err = badFileStore.DeleteBadFileByPath(path)
		if err != nil {
			t.Fatalf("Failed to delete bad file by path: %v", err)
		}

		// Verify it was deleted
		remainingFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get remaining bad files: %v", err)
		}

		for _, f := range remainingFiles {
			if f.Path == path {
				t.Error("Bad file was not deleted by path")
			}
		}
	})

	t.Run("DeleteBadFile_NonExistent", func(t *testing.T) {
		// Test deleting a non-existent bad file
		err := badFileStore.DeleteBadFile(99999)
		if err != nil {
			t.Fatalf("Expected no error when deleting non-existent bad file, got: %v", err)
		}

		err = badFileStore.DeleteBadFileByPath("/non/existent/path.cbz")
		if err != nil {
			t.Fatalf("Expected no error when deleting non-existent bad file by path, got: %v", err)
		}
	})

	t.Run("CountBadFiles", func(t *testing.T) {
		// Test counting bad files
		// First, clear existing bad files
		existingFiles, err := badFileStore.GetAllBadFiles()
		if err != nil {
			t.Fatalf("Failed to get existing bad files: %v", err)
		}

		for _, bf := range existingFiles {
			err := badFileStore.DeleteBadFile(bf.ID)
			if err != nil {
				t.Fatalf("Failed to delete existing bad file: %v", err)
			}
		}

		// Count should be 0
		count, err := badFileStore.CountBadFiles()
		if err != nil {
			t.Fatalf("Failed to count bad files: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0 bad files, got %d", count)
		}

		// Create some bad files
		paths := []string{
			"/test/path/count1.cbz",
			"/test/path/count2.cbz",
			"/test/path/count3.cbz",
		}

		for _, path := range paths {
			err := badFileStore.CreateBadFile(path, "test_error", 1024)
			if err != nil {
				t.Fatalf("Failed to create bad file %s: %v", path, err)
			}
		}

		// Count should be 3
		count, err = badFileStore.CountBadFiles()
		if err != nil {
			t.Fatalf("Failed to count bad files: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 bad files, got %d", count)
		}
	})

}

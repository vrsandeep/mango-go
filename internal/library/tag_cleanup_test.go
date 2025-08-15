// This file tests the tag cleanup functionality.

package library_test

import (
	"testing"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
	"github.com/vrsandeep/mango-go/internal/library"
	"github.com/vrsandeep/mango-go/internal/models"
	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// TestDeleteEmptyTags tests the DeleteEmptyTags functionality
func TestDeleteEmptyTags(t *testing.T) {
	app := testutil.SetupTestApp(t)
	st := store.New(app.DB())

	// Create some test tags by adding them to folders
	tag1 := &models.Tag{Name: "Action"}
	tag2 := &models.Tag{Name: "Adventure"}

	// Create test folders
	folder1, err := st.CreateFolder("/test/folder1", "Test Folder 1", nil)
	if err != nil {
		t.Fatalf("Failed to create folder1: %v", err)
	}
	folder2, err := st.CreateFolder("/test/folder2", "Test Folder 2", nil)
	if err != nil {
		t.Fatalf("Failed to create folder2: %v", err)
	}

	// Add tags to folders to create them
	_, err = st.AddTagToFolder(folder1.ID, tag1.Name)
	if err != nil {
		t.Fatalf("Failed to add tag1 to folder: %v", err)
	}
	_, err = st.AddTagToFolder(folder2.ID, tag2.Name)
	if err != nil {
		t.Fatalf("Failed to add tag2 to folder: %v", err)
	}

	// Verify tags were created
	tags, err := st.ListTagsWithCounts()
	if err != nil {
		t.Fatalf("Failed to get tags: %v", err)
	}
	if len(tags) < 2 {
		t.Fatalf("Expected at least 2 tags, got %d", len(tags))
	}

	// Run DeleteEmptyTags
	ctx := &testutil.MockJobContext{App: app}
	library.DeleteEmptyTags(ctx)

	// Verify that only tags with associations remain
	tags, err = st.ListTagsWithCounts()
	if err != nil {
		t.Fatalf("Failed to get tags after cleanup: %v", err)
	}

	// Check that tags with folder associations remain
	var foundTag1, foundTag2 bool
	for _, tag := range tags {
		if tag.Name == "Action" && tag.FolderCount > 0 {
			foundTag1 = true
		}
		if tag.Name == "Adventure" && tag.FolderCount > 0 {
			foundTag2 = true
		}
	}

	if !foundTag1 {
		t.Error("Tag 'Action' should remain after cleanup since it's associated with a folder")
	}
	if !foundTag2 {
		t.Error("Tag 'Adventure' should remain after cleanup since it's associated with a folder")
	}
}

// TestDeleteEmptyTagsWithNoEmptyTags tests the case where there are no empty tags to delete
func TestDeleteEmptyTagsWithNoEmptyTags(t *testing.T) {
	app := testutil.SetupTestApp(t)
	st := store.New(app.DB())

	// Create a test folder
	folder, err := st.CreateFolder("/test/folder", "Test Folder", nil)
	if err != nil {
		t.Fatalf("Failed to create folder: %v", err)
	}

	// Add a tag to the folder
	_, err = st.AddTagToFolder(folder.ID, "Test Tag")
	if err != nil {
		t.Fatalf("Failed to add tag to folder: %v", err)
	}

	// Get initial tag count
	initialTags, err := st.ListTagsWithCounts()
	if err != nil {
		t.Fatalf("Failed to get initial tags: %v", err)
	}
	initialCount := len(initialTags)

	// Run DeleteEmptyTags - should not delete anything since tag is associated
	ctx := &testutil.MockJobContext{App: app}
	library.DeleteEmptyTags(ctx)

	// Verify tag count remains the same
	finalTags, err := st.ListTagsWithCounts()
	if err != nil {
		t.Fatalf("Failed to get final tags: %v", err)
	}
	if len(finalTags) != initialCount {
		t.Errorf("Expected tag count to remain the same, got %d (was %d)", len(finalTags), initialCount)
	}
}

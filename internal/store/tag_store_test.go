package store_test

import (
	"testing"

	"github.com/vrsandeep/mango-go/internal/store"
	"github.com/vrsandeep/mango-go/internal/testutil"
)

// Run the whole test suite as it is a sequential test.
func TestAddTagToFolder(t *testing.T) {
	db := testutil.SetupTestDB(t)
	s := store.New(db)
	folder, _ := s.CreateFolder("/test", "Test Folder", nil)

	t.Run("Add Tag To Folder", func(t *testing.T) {

		// Add a tag to the folder
		tag, err := s.AddTagToFolder(folder.ID, "action")
		if err != nil {
			t.Fatalf("Failed to add tag to folder: %v", err)
		}
		if tag.Name != "action" {
			t.Errorf("Expected tag name 'action', got '%s'", tag.Name)
		}

		// Add the same tag again (should not error, should not duplicate)
		tag2, err := s.AddTagToFolder(folder.ID, "action")
		if err != nil {
			t.Fatalf("Failed to add duplicate tag to folder: %v", err)
		}
		if tag2.ID != tag.ID {
			t.Errorf("Expected same tag ID, got %d and %d", tag.ID, tag2.ID)
		}
	})

	t.Run("Get Tag By ID", func(t *testing.T) {
		tag, err := s.GetTagByID(1)
		if err != nil {
			t.Fatalf("Failed to get tag by ID: %v", err)
		}
		if tag.Name != "action" {
			t.Errorf("Expected tag name 'action', got '%s'", tag.Name)
		}
	})

	t.Run("Remove Tag From Folder", func(t *testing.T) {
		err := s.RemoveTagFromFolder(folder.ID, 1)
		if err != nil {
			t.Fatalf("Failed to remove tag from folder: %v", err)
		}

		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM folder_tags WHERE folder_id = ? AND tag_id = ?", folder.ID, 1).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to get count of tags in folder_tags: %v", err)
		}
		if count != 0 {
			t.Errorf("Expected 0 tags in folder_tags, got %d", count)
		}
	})
}
